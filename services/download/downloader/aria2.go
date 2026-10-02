package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"maps"
	"mime"
	"net/http"
	"os/exec"
	"path"
	"path/filepath"
	"rapid/lib"
	"rapid/lib/errors"
	"rapid/services/download/api"
	"rapid/services/download/rpc"
	rpcapi "rapid/services/download/rpc/api"
	"rapid/services/settings"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/sync/errgroup"
)

// JSONRPCVersion is the aria2 JSON-RPC version.
const JSONRPCVersion = "2.0"

// StatusKeys are the tellStatus fields fetched for status updates.
var StatusKeys = []string{
	"completedLength",
	"connections",
	"dir",
	"downloadSpeed",
	"errorCode",
	"errorMessage",
	"files",
	"gid",
	"numPieces",
	"pieceLength",
	"status",
	"totalLength",
	"verifiedLength",
}

// WSEvents are the aria2 websocket notifications we react to.
var WSEvents = map[string]bool{
	"aria2.onDownloadComplete": true,
	"aria2.onDownloadError":    true,
}

// validSchemes preserves the loose Python substring semantics (do NOT "fix"
// to scheme parsing; resolver tests depend on it).
var validSchemes = []string{"http", "https", "ftp", "ftps"}

// contextKeys are browser-context option keys that never reach aria2.
var contextKeys = map[string]bool{
	"headers":  true,
	"cookies":  true,
	"referer":  true,
	"pageUrl":  true,
	"title":    true,
	"filename": true,
	"mimeType": true,
	"source":   true,
}

type listenerEntry struct {
	onNotify NotifyCallback
	onError  ErrorCallback
}

// Aria2Downloader is a Downloader+Resolver backed by a local aria2c daemon.
// All callbacks carry typed structs; Refresh on a stopped downloader still
// works (the poll loop's ctx cancellation is the lifecycle control).
type Aria2Downloader struct {
	settings     settings.Settings
	manageDaemon bool
	globalNotify GlobalNotifyCallback
	rpc          rpc.Rpc
	resolverSem  chan struct{}

	mu            sync.Mutex
	running       bool
	process       *exec.Cmd
	ownsDaemon    bool
	lastSpawn     time.Time
	listening     map[string]listenerEntry
	wsConn        *websocket.Conn
	wsCancel      context.CancelFunc
	wsDone        chan struct{}
	pendingCancel context.CancelFunc
	pendingGen    int
}

// NewAria2Downloader builds a downloader; rpc may be nil (uses Aria2RPC).
func NewAria2Downloader(s settings.Settings, rpc rpc.Rpc, manageDaemon bool, onGlobalNotify GlobalNotifyCallback) *Aria2Downloader {
	return &Aria2Downloader{
		settings:     s,
		rpc:          rpc,
		manageDaemon: manageDaemon,
		globalNotify: onGlobalNotify,
		resolverSem:  make(chan struct{}, 2),
		listening:    map[string]listenerEntry{},
	}
}

func (d *Aria2Downloader) spawnDaemon(ctx context.Context) error {
	program, err := exec.LookPath("aria2c")
	if err != nil || program == "" {
		return errors.NewAria2Error("aria2c binary not found")
	}
	d.mu.Lock()
	alive := isAlive(d.process)
	d.mu.Unlock()
	if alive {
		return nil
	}

	// Adopt a daemon already serving our RPC port (e.g. an orphan left by a
	// previous crashed run) instead of spawning a second one.
	if _, err := d.rpc.Call(ctx, "aria2.getVersion", nil); err == nil {
		d.mu.Lock()
		d.ownsDaemon = false
		d.mu.Unlock()
		return nil
	}

	args := []string{
		"--enable-rpc",
		fmt.Sprintf("--rpc-listen-port=%d", d.settings.Aria2Port),
		fmt.Sprintf("--dir=%s", d.settings.DownloadDir),
		fmt.Sprintf("--save-session=%s", d.settings.Aria2SessionFile),
		fmt.Sprintf("--input-file=%s", d.settings.Aria2SessionFile),
	}
	if d.settings.Aria2Token != "" {
		args = append(args, "--rpc-secret="+d.settings.Aria2Token)
	}
	cmd := exec.CommandContext(ctx, program, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	d.mu.Lock()
	d.process = cmd
	d.mu.Unlock()

	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	deadline := time.Now().Add(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			if _, err := d.rpc.Call(ctx, "aria2.getVersion", nil); err == nil {
				d.mu.Lock()
				d.ownsDaemon = true
				d.mu.Unlock()
				return nil
			}

			if !isAlive(cmd) {
				d.mu.Lock()
				d.process = nil
				d.mu.Unlock()
				return errors.NewAria2Error("aria2 exited during startup")
			}

			if time.Now().After(deadline) {
				d.mu.Lock()
				d.process = nil
				d.mu.Unlock()
				return errors.NewAria2Error("aria2 did not become ready on port " + strconv.Itoa(d.settings.Aria2Port))
			}
		}
	}
}

func (d *Aria2Downloader) ensureDaemon(ctx context.Context) {
	if !d.manageDaemon {
		return
	}
	d.mu.Lock()
	owns := d.ownsDaemon
	alive := isAlive(d.process)
	since := time.Since(d.lastSpawn)
	if owns && !alive && since >= 5*time.Second {
		d.lastSpawn = time.Now()
	}
	should := owns && !alive && since >= 5*time.Second
	d.mu.Unlock()
	if !should {
		return
	}

	d.spawnDaemon(ctx)
}

func (d *Aria2Downloader) wsURL() string {
	u := fmt.Sprintf("ws://%s:%d/jsonrpc", d.settings.Aria2Host, d.settings.Aria2Port)
	if d.settings.Aria2Token != "" {
		u += "?token=" + d.settings.Aria2Token
	}
	return u
}

func (d *Aria2Downloader) wsRun(ctx context.Context) {
	for {
		conn, _, err := websocket.Dial(ctx, d.wsURL(), nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			d.ensureDaemon(ctx)
			log.Print("websocket connect failed, retrying")
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		d.mu.Lock()
		d.wsConn = conn
		d.mu.Unlock()
		d.recvLoop(ctx, conn)
		if ctx.Err() != nil {
			_ = conn.Close(websocket.StatusGoingAway, "shutdown")
		} else {
			_ = conn.Close(websocket.StatusNormalClosure, "")
		}

		d.mu.Lock()
		if d.wsConn == conn {
			d.wsConn = nil
		}
		d.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (d *Aria2Downloader) killWait(proc *exec.Cmd) error {
	_ = proc.Process.Kill()
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- proc.Wait()
	}()

	select {
	case <-waitDone:
		return nil
	case <-time.After(2 * time.Second):
		return <-waitDone
	}
}

func (d *Aria2Downloader) recvLoop(ctx context.Context, conn *websocket.Conn) {
	for {
		_, msg, err := conn.Read(ctx)
		if err != nil {
			return
		}
		d.onWsMessage(ctx, msg)
	}
}

func (d *Aria2Downloader) onWsMessage(ctx context.Context, msg []byte) {
	var n rpcapi.Aria2WebsocketNotification
	if err := json.Unmarshal(msg, &n); err != nil {
		return
	}
	if !WSEvents[n.Method] {
		return
	}

	for _, param := range n.Params {
		d.Refresh(ctx, param.GID)
	}
}

// Start spawns the daemon when managed and launches the websocket goroutine.
// A missing daemon degrades (logged) instead of failing
func (d *Aria2Downloader) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = true
	d.mu.Unlock()

	if d.manageDaemon {
		if err := d.spawnDaemon(ctx); err != nil {
			log.Printf("aria2: %v (continuing degraded)", err)
		}
	}

	wsCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	d.mu.Lock()
	d.wsCancel = cancel
	d.wsDone = done
	d.mu.Unlock()

	go func() {
		defer close(done)
		d.wsRun(wsCtx)
	}()

	return nil
}

// Stop cancels the ws loop and joins it, then terminates the daemon only when
// we own it.
func (d *Aria2Downloader) Stop() error {
	d.mu.Lock()
	d.running = false
	d.listening = map[string]listenerEntry{}
	cancel := d.wsCancel
	conn := d.wsConn
	done := d.wsDone
	d.wsCancel = nil
	d.wsConn = nil
	d.wsDone = nil
	proc := d.process
	owns := d.ownsDaemon
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if conn != nil {
		_ = conn.Close(websocket.StatusGoingAway, "shutdown")
	}
	if done != nil {
		<-done
	}
	if proc == nil || !owns {
		return nil
	}
	d.mu.Lock()
	d.process = nil
	d.mu.Unlock()
	if err := proc.Process.Signal(syscall.SIGTERM); err != nil {
		return d.killWait(proc)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- proc.Wait()
	}()

	select {
	case <-waitDone:
		return nil
	case <-time.After(2 * time.Second):
		return d.killWait(proc)
	}
}

// ShouldResolve preserves the loose substring semantics.
func (d *Aria2Downloader) ShouldResolve(uri string) bool {
	for _, pat := range validSchemes {
		if ok, _ := regexp.MatchString(pat, uri); ok {
			return true
		}
	}
	return false
}

// finalURL picks the used uri, else the first, else the fallback.
func (d *Aria2Downloader) finalURL(file api.DownloadFile, fallback string) string {
	if len(file.URIs) == 0 {
		return fallback
	}
	for _, u := range file.URIs {
		if u.Status == "used" && u.URI != "" {
			return u.URI
		}
	}
	if file.URIs[0].URI != "" {
		return file.URIs[0].URI
	}
	return fallback
}

// probeHeader is a best-effort HEAD probe; any failure yields nil.
func (d *Aria2Downloader) probeHeader(ctx context.Context, rawURL string, headers map[string]string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	out := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		if len(v) > 0 {
			out[strings.ToLower(k)] = v[0]
		}
	}
	return out, nil
}

func (d *Aria2Downloader) pollStatus(ctx context.Context, gid string) (api.Download, error) {
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	timeout := time.After(2 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return api.Download{}, ctx.Err()
		case <-tick.C:
			status, err := d.GetStatus(ctx, gid)
			if err != nil {
				return api.Download{}, err
			}

			s := status.Status
			if s == "complete" || s == "error" {
				return status, nil
			}
		case <-timeout:
			return d.GetStatus(ctx, gid)
		}
	}
}

// doRequest runs addUri and the HEAD probe concurrently.
func (d *Aria2Downloader) doRequest(ctx context.Context, uri string, options map[string]any, requestHeaders map[string]string) (string, map[string]string, error) {
	g, gCtx := errgroup.WithContext(ctx)

	var gid string
	var headers map[string]string

	g.Go(func() error {
		b, err := json.Marshal([]any{[]string{uri}, options})
		if err != nil {
			return err
		}

		raw, err := d.rpc.Call(gCtx, "aria2.addUri", b)
		if err != nil {
			return err
		}

		var res rpcapi.Aria2AddUriResponse
		if err := json.Unmarshal(raw, &res); err != nil || res.Result == "" {
			return err
		}

		gid = res.Result
		return nil
	})

	g.Go(func() error {
		var err error
		headers, err = d.probeHeader(gCtx, uri, requestHeaders)
		if err != nil {
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return "", nil, err
	}

	return gid, headers, nil
}

// Resolve dry-runs uri through aria2 plus a HEAD probe and returns metadata.
// Each new Resolve cancels the previous resolve's context (replacing
// _pendingResolveFutures).
func (d *Aria2Downloader) Resolve(ctx context.Context, uri string, options map[string]any) ([]api.ResolvedURL, error) {
	select {
	case d.resolverSem <- struct{}{}:
		defer func() { <-d.resolverSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	d.mu.Lock()
	if d.pendingCancel != nil {
		d.pendingCancel()
	}
	rctx, cancel := context.WithCancel(ctx)
	d.pendingCancel = cancel
	d.pendingGen++
	gen := d.pendingGen
	d.mu.Unlock()
	defer func() {
		cancel()
		d.mu.Lock()
		if d.pendingGen == gen {
			d.pendingCancel = nil
		}
		d.mu.Unlock()
	}()

	if options == nil {
		options = map[string]any{}
	}
	rawHeaders, _ := options["headers"].(map[string]string)
	headers := maps.Clone(rawHeaders)

	rawCookies, _ := options["cookies"].(map[string]string)
	cookies := maps.Clone(rawCookies)

	referer := ""
	if s, _ := options["referer"].(string); s != "" {
		referer = s
	} else if s, _ := options["pageUrl"].(string); s != "" {
		referer = s
	}
	requestHeaders := maps.Clone(headers)
	if referer != "" {
		requestHeaders["Referer"] = referer
	}
	if len(cookies) > 0 {
		parts := make([]string, 0, len(cookies))
		for k, v := range cookies {
			parts = append(parts, k+"="+v)
		}
		requestHeaders["Cookie"] = strings.Join(parts, "; ")
	}

	ariaOptions := map[string]any{}
	for k, v := range options {
		if !contextKeys[k] {
			ariaOptions[k] = v
		}
	}

	maps.Copy(ariaOptions, d.aria2Options(api.ResolvedURL{URL: uri, Headers: headers, Cookies: cookies, Referer: referer}))
	ariaOptions["dry-run"] = "true"
	ariaOptions["use-head"] = "true"

	gid, respHeaders, err := d.doRequest(rctx, uri, ariaOptions, requestHeaders)
	if err != nil {
		return nil, err
	}
	defer d.onResolved(rctx, gid)

	status, err := d.pollStatus(rctx, gid)
	if err != nil {
		return nil, err
	}
	var fileInfo api.DownloadFile
	if len(status.Files) > 0 {
		fileInfo = status.Files[0]
	}
	filename := ""
	if fileInfo.Path != "" {
		filename = filepath.Base(fileInfo.Path)
	} else {
		filename = filenameOf(uri)
	}

	final := d.finalURL(fileInfo, uri)

	var mimeType string
	if respHeaders != nil {
		mimeType = respHeaders["content-type"]
	}
	if mimeType == "" {
		mimeType = mime.TypeByExtension(path.Ext(filename))
	}
	var contentLength int64
	if respHeaders != nil {
		if cl := respHeaders["content-length"]; cl != "" {
			contentLength = lib.StringToInt[int64](cl)
		}
	}
	if contentLength == 0 {
		contentLength = fileInfo.Length
	}
	if mimeType != "" && filename != "" && filename != "." {
		suffix := path.Ext(filename)
		if suffix == "" || !lib.IsAlnum(suffix[1:]) {
			if ext := lib.ExtensionForType(mimeType); ext != "" {
				filename += ext
			}
		}
	}

	return []api.ResolvedURL{{
		URL:          final,
		Title:        filename,
		Filename:     filename,
		MIMEType:     mimeType,
		Size:         contentLength,
		Category:     getCategory(mimeType, filename),
		Headers:      headers,
		Cookies:      cookies,
		Referer:      referer,
		ResolverName: "Rapid",
	}}, nil
}

// onResolved halts the dry-run download, then retries removeDownloadResult 5x.
func (d *Aria2Downloader) onResolved(ctx context.Context, gid string) {
	b, err := json.Marshal([]string{gid})
	if err != nil {
		return
	}

	_, _ = d.rpc.Call(ctx, "aria2.remove", b)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	timeout := time.After(time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			b, err := json.Marshal([]string{gid})
			if err != nil {
				return
			}

			if _, err := d.rpc.Call(ctx, "aria2.removeDownloadResult", b); err == nil {
				return
			}
		case <-timeout:
			log.Printf("failed to remove resolve GID %s", gid)
			return
		}
	}
}

// GetStatus unmarshals tellStatus straight into Download.
func (d *Aria2Downloader) GetStatus(ctx context.Context, id string) (api.Download, error) {
	b, err := json.Marshal([]any{id, StatusKeys})
	if err != nil {
		return api.Download{}, err
	}

	raw, err := d.rpc.Call(ctx, "aria2.tellStatus", b)
	if err != nil {
		return api.Download{}, err
	}

	var status rpcapi.Aria2TellStatusResponse
	if err := json.Unmarshal(raw, &status); err != nil {
		return api.Download{}, err
	}

	return status.Result.ToDownload(), nil
}

func (d *Aria2Downloader) aria2Options(r api.ResolvedURL) map[string]any {
	options := map[string]any{}
	headers := maps.Clone(r.Headers)
	if r.Filename != "" {
		options["out"] = r.Filename
	}
	if r.Dir != "" {
		options["dir"] = r.Dir
	}
	if r.Referer != "" {
		headers["Referer"] = r.Referer
	}
	if len(headers) > 0 {
		hs := make([]string, 0, len(headers))
		for k, v := range headers {
			hs = append(hs, k+": "+v)
		}
		options["header"] = hs
	}
	if len(r.Cookies) > 0 {
		parts := make([]string, 0, len(r.Cookies))
		for k, v := range r.Cookies {
			parts = append(parts, k+"="+v)
		}
		hdrs, _ := options["header"].([]string)
		options["header"] = append(hdrs, "Cookie: "+strings.Join(parts, "; "))
	}
	return options
}

// Download starts a download and returns its initial state.
func (d *Aria2Downloader) Download(ctx context.Context, uri api.ResolvedURL) (api.Download, error) {
	options := d.aria2Options(uri)
	b, err := json.Marshal([]any{[]string{uri.URL}, options})
	if err != nil {
		return api.Download{}, err
	}

	raw, err := d.rpc.Call(ctx, "aria2.addUri", b)
	if err != nil {
		return api.Download{}, err
	}

	var res rpcapi.Aria2AddUriResponse
	if err := json.Unmarshal(raw, &res); err != nil || res.Result == "" {
		return api.Download{}, err
	}
	return d.GetStatus(ctx, res.Result)
}

// Pause pauses and returns fresh status.
func (d *Aria2Downloader) Pause(ctx context.Context, id string) (api.Download, error) {
	b, err := json.Marshal([]any{id})
	if err != nil {
		return api.Download{}, err
	}

	if _, err := d.rpc.Call(ctx, "aria2.pause", b); err != nil {
		return api.Download{}, err
	}
	return d.GetStatus(ctx, id)
}

// Resume unpauses and returns fresh status.
func (d *Aria2Downloader) Resume(ctx context.Context, id string) (api.Download, error) {
	b, err := json.Marshal([]any{id})
	if err != nil {
		return api.Download{}, err
	}

	if _, err := d.rpc.Call(ctx, "aria2.unpause", b); err != nil {
		return api.Download{}, err
	}
	return d.GetStatus(ctx, id)
}

// Remove halts (active) then clears the result, keeping status via GetStatus.
func (d *Aria2Downloader) Remove(ctx context.Context, id string) (api.Download, error) {
	d.removeAny(ctx, id, "aria2.remove", "aria2.removeDownloadResult")
	return d.GetStatus(ctx, id)
}

// Purge clears the result first (order differs from Remove — keep it).
func (d *Aria2Downloader) Purge(ctx context.Context, id string) error {
	d.removeAny(ctx, id, "aria2.removeDownloadResult", "aria2.remove")
	return nil
}

func (d *Aria2Downloader) removeAny(ctx context.Context, id string, methods ...string) {
	for _, m := range methods {
		b, err := json.Marshal([]any{id})
		if err != nil {
			continue
		}

		if _, err := d.rpc.Call(ctx, m, b); err == nil {
			return
		}
	}
}

// Listen registers callbacks (mutex-guarded, no I/O).
func (d *Aria2Downloader) Listen(id string, onNotify NotifyCallback, onError ErrorCallback) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listening[id] = listenerEntry{
		onNotify: onNotify,
		onError:  onError,
	}
}

// Unlisten drops callbacks for id.
func (d *Aria2Downloader) Unlisten(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.listening, id)
}

func (d *Aria2Downloader) notify(ctx context.Context, gid string, e listenerEntry) {
	status, err := d.GetStatus(ctx, gid)
	if err != nil {
		e.onError(err)
		d.Unlisten(gid)
		return
	}
	e.onNotify(status)
	if s := status.Status; s == "error" || s == "removed" || s == "complete" {
		d.Unlisten(gid)
	}
}

func (d *Aria2Downloader) Refresh(ctx context.Context, id ...string) {
	for _, gid := range id {
		if e, ok := d.listening[gid]; ok {
			d.notify(ctx, gid, e)
		}
	}

	if len(id) > 0 {
		return
	}

	res, err := d.rpc.Call(ctx, "aria2.getGlobalStat", nil)
	if err != nil {
		log.Print(err)
		return
	}

	if d.globalNotify != nil {
		d.globalNotify(res)
	}
}
