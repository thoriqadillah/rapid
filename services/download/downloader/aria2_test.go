package downloader

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"rapid/services/download/api"
	"rapid/services/download/rpc"
	"rapid/services/settings"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSettings(dir string) settings.Settings {
	s := settings.Settings{
		DataDir:          dir,
		DownloadDir:      dir,
		BaseDir:          dir,
		Aria2Host:        "127.0.0.1",
		Aria2Port:        6800,
		Aria2SessionFile: dir + "/aria2.session",
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	if f, err := os.OpenFile(s.Aria2SessionFile, os.O_CREATE|os.O_APPEND, 0o644); err == nil {
		_ = f.Close()
	}
	return s
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitPort(t *testing.T, port int) {
	t.Helper()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
			if err == nil {
				_ = c.Close()
				return
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("port %d never opened", port)
		}
	}
}

func waitUntil(t *testing.T, pred func() bool, timeout time.Duration) {
	t.Helper()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if pred() {
				return
			}
		case <-time.After(timeout):
			t.Fatal("condition not met before timeout")
		}
	}
}

func serveDir(t *testing.T, dir string) (string, func()) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: http.FileServer(http.Dir(dir))}
	go func() { srv.Serve(l) }()
	return "http://" + l.Addr().String(), func() { _ = srv.Close() }
}

func spawnReal(t *testing.T, dir string, notify ...GlobalNotifyCallback) *Aria2Downloader {
	t.Helper()
	if _, err := exec.LookPath("aria2c"); err != nil {
		t.Skip("aria2c not installed")
	}
	dlDir := filepath.Join(dir, "dl")
	require.NoError(t, os.MkdirAll(dlDir, 0o755))
	s := testSettings(dir)
	s.DownloadDir = dlDir
	s.Aria2Port = freePort(t)
	var onNotify GlobalNotifyCallback
	if len(notify) > 0 {
		onNotify = notify[0]
	}
	rpc := rpc.NewJSONRpc(s.Aria2Host, s.Aria2Port, s.Aria2Token)
	dl := NewAria2Downloader(s, rpc, true, onNotify)
	require.NoError(t, dl.Start(t.Context()))

	waitPort(t, s.Aria2Port)
	t.Cleanup(func() { dl.Stop() })

	return dl
}

func writePayload(t *testing.T, dir, name string, size int, fill byte) {
	t.Helper()
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = fill
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), payload, 0o644))
}

func TestIntegrationResolveReturnsMetadata(t *testing.T) {
	dir := t.TempDir()
	dl := spawnReal(t, dir)
	writePayload(t, dir, "clip.mp4", 1024, 'x')
	base, closeSrv := serveDir(t, dir)
	defer closeSrv()

	items, err := dl.Resolve(t.Context(), base+"/clip.mp4", nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "clip.mp4", items[0].Filename)
	assert.Equal(t, int64(1024), items[0].Size)
	assert.Equal(t, "video", items[0].Category.String())
	assert.Equal(t, "video/mp4", items[0].MIMEType)
}

func TestIntegrationDownloadCompletes(t *testing.T) {
	dir := t.TempDir()
	dl := spawnReal(t, dir)
	writePayload(t, dir, "payload.bin", 512*1024, 'y')
	base, closeSrv := serveDir(t, dir)
	defer closeSrv()

	d, err := dl.Download(t.Context(), api.ResolvedURL{URL: base + "/payload.bin"})
	require.NoError(t, err)

	waitUntil(t, func() bool {
		s, err := dl.GetStatus(t.Context(), d.GID)
		return err == nil && s.Status == "complete"
	}, 30*time.Second)

	s, err := dl.GetStatus(t.Context(), d.GID)
	require.NoError(t, err)
	assert.Equal(t, 1.0, s.Progress())
	_, err = os.Stat(filepath.Join(dl.settings.DownloadDir, "payload.bin"))
	assert.NoError(t, err, "file must exist")
}

func TestIntegrationRefreshPushesStatus(t *testing.T) {
	dir := t.TempDir()
	dl := spawnReal(t, dir)
	writePayload(t, dir, "note.bin", 256*1024, 'z')
	base, closeSrv := serveDir(t, dir)
	defer closeSrv()

	d, err := dl.Download(t.Context(), api.ResolvedURL{URL: base + "/note.bin"})
	require.NoError(t, err)
	var mu sync.Mutex
	var statuses []string

	dl.Listen(d.GID,
		func(s api.Download) {
			mu.Lock()
			defer mu.Unlock()
			if s.Status != "" {
				statuses = append(statuses, s.Status)
			}
		},
		func(error) {})

	waitUntil(t, func() bool {
		dl.Refresh(t.Context())
		mu.Lock()
		defer mu.Unlock()
		return slices.Contains(statuses, "complete")
	}, 30*time.Second)

	assert.Len(t, statuses, 1)
}

// A WS notification without a usable gid must not take the process down
// (aria2c can't emit malformed notifications on demand, so the bytes are
// hand-crafted against a live downloader).
func TestOnWsMessageWithoutGidDoesNotPanic(t *testing.T) {
	dl := spawnReal(t, t.TempDir())
	for _, msg := range []string{
		`{"method":"aria2.onDownloadComplete","params":[]}`,
		`{"method":"aria2.onDownloadComplete"}`,
		`{"method":"aria2.onDownloadComplete","params":[42]}`,
	} {
		assert.NotPanics(t, func() {
			dl.onWsMessage(t.Context(), []byte(msg))
		}, "message: %s", msg)
	}
}

func TestGetStatusSurfacesDaemonError(t *testing.T) {
	dl := spawnReal(t, t.TempDir())
	_, err := dl.GetStatus(t.Context(), "dead-gid")
	require.Error(t, err, "daemon error reply must be a Go error")
	assert.EqualError(t, err, "aria2 error code 1: Invalid GID dead-gid")
}

// serveThrottled serves payload capped at bytesPerSecond so downloads stay
// active long enough to pause or observe websocket pushes. HEAD requests get
// headers only (no pacing); aria2 opens a single connection by default, so
// no Range handling is needed.
func serveThrottled(t *testing.T, payload []byte, bytesPerSecond int, contentType string) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		start, end := 0, len(payload)-1
		status := http.StatusOK
		// aria2 resumes with Range (pause/resume, multi-connection retry);
		// answering 200-from-zero there makes it refetch forever.
		if rng := r.Header.Get("Range"); strings.HasPrefix(rng, "bytes=") {
			if a, b, ok := strings.Cut(strings.TrimPrefix(rng, "bytes="), "-"); ok {
				if s, err := strconv.Atoi(strings.TrimSpace(a)); err == nil && s >= 0 {
					start = s
				}
				if e, err := strconv.Atoi(strings.TrimSpace(b)); err == nil && e >= start {
					end = e
				}
			}
			if start < len(payload) {
				end = min(end, len(payload)-1)
				status = http.StatusPartialContent
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(payload)))
				w.Header().Set("Accept-Ranges", "bytes")
			}
		}
		w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
		if r.Method == http.MethodHead {
			return
		}
		w.WriteHeader(status)
		chunk := bytesPerSecond / 10
		if chunk < 1 {
			chunk = 1
		}
		for i := start; i <= end; i += chunk {
			if _, err := w.Write(payload[i:min(i+chunk, end+1)]); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(100 * time.Millisecond)
		}
	})}
	go func() { _ = srv.Serve(l) }()
	t.Cleanup(func() { _ = srv.Close() })
	return "http://" + l.Addr().String()
}

func makePayload(size int, fill byte) []byte {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = fill
	}
	return payload
}

func TestIntegrationPauseResume(t *testing.T) {
	dir := t.TempDir()
	dl := spawnReal(t, dir)
	base := serveThrottled(t, makePayload(1<<20, 'p'), 256<<10, "application/octet-stream")

	d, err := dl.Download(t.Context(), api.ResolvedURL{URL: base + "/slow.bin"})
	require.NoError(t, err)
	waitUntil(t, func() bool {
		s, err := dl.GetStatus(t.Context(), d.GID)
		return err == nil && s.CompletedLength > 0
	}, 15*time.Second)

	_, err = dl.Pause(t.Context(), d.GID)
	require.NoError(t, err)
	waitUntil(t, func() bool {
		s, err := dl.GetStatus(t.Context(), d.GID)
		return err == nil && s.Status == "paused"
	}, 15*time.Second)

	_, err = dl.Resume(t.Context(), d.GID)
	require.NoError(t, err)
	waitUntil(t, func() bool {
		s, err := dl.GetStatus(t.Context(), d.GID)
		return err == nil && s.Status == "complete"
	}, 60*time.Second)
}

func TestIntegrationStopRetainsRemoved(t *testing.T) {
	dir := t.TempDir()
	dl := spawnReal(t, dir)
	base := serveThrottled(t, makePayload(1<<20, 'g'), 256<<10, "application/octet-stream")

	d, err := dl.Download(t.Context(), api.ResolvedURL{URL: base + "/gone.bin"})
	require.NoError(t, err)
	waitUntil(t, func() bool {
		s, err := dl.GetStatus(t.Context(), d.GID)
		return err == nil && s.Status == "active"
	}, 15*time.Second)

	// Stopping an active download retains "removed" (stopping a completed
	// one instead purges the result, and GetStatus rightly errors).
	stopped, err := dl.Remove(t.Context(), d.GID)
	require.NoError(t, err)
	assert.Equal(t, "removed", stopped.Status)
	s, err := dl.GetStatus(t.Context(), d.GID)
	require.NoError(t, err)
	assert.Equal(t, "removed", s.Status, "removed state must be retained")
}

func TestIntegrationPurgeForgetsGid(t *testing.T) {
	dir := t.TempDir()
	dl := spawnReal(t, dir)
	writePayload(t, dir, "temp.bin", 1024, 't')
	base, closeSrv := serveDir(t, dir)
	defer closeSrv()

	d, err := dl.Download(t.Context(), api.ResolvedURL{URL: base + "/temp.bin"})
	require.NoError(t, err)
	waitUntil(t, func() bool {
		s, err := dl.GetStatus(t.Context(), d.GID)
		return err == nil && s.Status == "complete"
	}, 30*time.Second)

	require.NoError(t, dl.Purge(t.Context(), d.GID))
	_, err = dl.GetStatus(t.Context(), d.GID)
	require.Error(t, err, "purged gid must be forgotten")
}

func TestIntegrationWebsocketPushesStatus(t *testing.T) {
	dir := t.TempDir()
	dl := spawnReal(t, dir)
	base := serveThrottled(t, makePayload(1<<19, 'w'), 256<<10, "application/octet-stream")

	d, err := dl.Download(t.Context(), api.ResolvedURL{URL: base + "/pushed.bin"})
	require.NoError(t, err)
	var mu sync.Mutex
	var statuses []string
	dl.Listen(d.GID,
		func(s api.Download) {
			mu.Lock()
			defer mu.Unlock()
			statuses = append(statuses, s.Status)
		},
		func(error) {})

	// NOTE: no Refresh calls — completion must arrive via websocket push.
	waitUntil(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return slices.Contains(statuses, "complete")
	}, 30*time.Second)
}

func TestIntegrationGlobalStatNotify(t *testing.T) {
	dir := t.TempDir()
	var mu sync.Mutex
	var got []byte
	dl := spawnReal(t, dir, func(b []byte) {
		mu.Lock()
		defer mu.Unlock()
		got = b
	})

	dl.Refresh(t.Context())
	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, got)
	var env struct {
		Result map[string]any `json:"result"`
	}
	require.NoError(t, json.Unmarshal(got, &env))
	assert.Contains(t, env.Result, "numActive")
}
