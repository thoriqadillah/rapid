// Package browser is the local, extension-only bridge that forwards browser
// requests (typed BrowserRequest, not dicts) to the download service.
// Served by Echo with CORS + body-limit middleware and a JSON error shape.
package browser

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"rapid/lib/errors"
)

// MaxBodyBytes caps extension request bodies at 2MiB.
const MaxBodyBytes = 2 * 1024 * 1024

// AllowedOriginSchemes are the extension origin prefixes (prefix check, NOT
// url parsing — mirrors Python).
var AllowedOriginSchemes = []string{"chrome-extension://", "moz-extension://"}

// BlockedHeaders are never inherited from the browser context (hop-by-hop or
// identity headers must not be replayed; see package notes in the reply).
var BlockedHeaders = map[string]bool{
	"connection":       true,
	"content-length":   true,
	"cookie":           true,
	"host":             true,
	"proxy-connection": true,
}

// DefaultAddr is the loopback bridge address.
const DefaultAddr = "127.0.0.1:17654"

// BrowserRequest is the typed bridge payload, decoded straight from the
// extension JSON (replaces the dict that flowed extension -> normalize ->
// downloadRequested -> resolveRequest). Size stays a pointer (absent vs 0
// matters to _browserResolvedURL).
type BrowserRequest struct {
	URL             string         `json:"url"`
	Headers         map[string]any `json:"headers"`
	Cookies         map[string]any `json:"cookies"`
	Referer         string         `json:"referer"`
	PageURL         string         `json:"pageUrl"`
	Title           string         `json:"title"`
	Filename        string         `json:"filename"`
	MIMEType        string         `json:"mimeType"`
	Category        string         `json:"category"`
	Source          string         `json:"source"`
	SavePath        string         `json:"savePath"`
	Size            *int64         `json:"size"`
	BrowserResolved bool           `json:"browserResolved"`
}

// Validate checks the URL scheme and strips blocked headers; a negative size
// is dropped (absent), like Python ignoring non-int sizes.
func (r *BrowserRequest) Validate() error {
	parsed, err := url.Parse(r.URL)
	if r.URL == "" || err != nil {
		return errors.NewBrowserRequestError("A downloadable HTTP, HTTPS, FTP, or FTPS URL is required")
	}
	switch parsed.Scheme {
	case "http", "https", "ftp", "ftps":
	default:
		return errors.NewBrowserRequestError("A downloadable HTTP, HTTPS, FTP, or FTPS URL is required")
	}
	for k := range r.Headers {
		if BlockedHeaders[strings.ToLower(k)] {
			delete(r.Headers, k)
		}
	}
	if r.Size != nil && *r.Size < 0 {
		r.Size = nil
	}
	return nil
}

func originAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, scheme := range AllowedOriginSchemes {
		if strings.HasPrefix(origin, scheme) {
			return true
		}
	}
	return false
}

// Config tunes the bridge; DefaultConfig mirrors the Python bridge.
type Config struct {
	Host            string
	Port            int
	MaxBodyBytes    int64
	ShutdownTimeout time.Duration
}

// DefaultConfig returns the stock loopback bridge configuration.
func DefaultConfig() Config {
	return Config{
		Host:            "127.0.0.1",
		Port:            17654,
		MaxBodyBytes:    MaxBodyBytes,
		ShutdownTimeout: 2 * time.Second,
	}
}

type BridgeResponse map[string]any

func errorResponse(msg string) BridgeResponse {
	return BridgeResponse{
		"ok":    false,
		"error": msg,
	}
}

// jsonErrorHandler keeps every framework error (405, 413, panics recovered
// by middleware) in the {ok:false,error} extension contract shape.
func jsonErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	msg := err.Error()
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		switch m := he.Message.(type) {
		case string:
			if m != "" {
				msg = m
			} else {
				msg = http.StatusText(code)
			}
		default:
			msg = http.StatusText(code)
		}
	}
	if c.Response().Committed {
		return
	}
	_ = c.JSON(code, errorResponse(msg))
}

// corsMiddleware echoes allowed Origins plus the fixed Allow headers, so
// handlers stay focused on routing.
func corsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		h := c.Response().Header()
		if origin := c.Request().Header.Get("Origin"); origin != "" && originAllowed(origin) {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Vary", "Origin")
		}
		h.Set("Access-Control-Allow-Headers", "Content-Type, X-Rapid-Extension")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		return next(c)
	}
}

func trustedMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		r := c.Request()
		trusted := originAllowed(r.Header.Get("Origin")) && r.Header.Get("X-Rapid-Extension") == "1"
		if !trusted {
			return c.JSON(http.StatusForbidden, errorResponse("Untrusted extension request"))
		}
		return next(c)
	}
}

func allowedOrigin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !originAllowed(c.Request().Header.Get("Origin")) ||
			!strings.Contains(strings.ToLower(c.Request().Header.Get("Access-Control-Request-Headers")), "x-rapid-extension") {
			return c.JSON(http.StatusForbidden, errorResponse("Untrusted extension origin"))
		}

		return next(c)
	}
}

// Bridge is the HTTP server the extension talks to. Start is once; bind
// errors return instead of emitting signals. Runtime serve-loop failures go
// to OnError.
type Bridge struct {
	cfg Config

	mu     sync.Mutex
	server *http.Server
	ln     net.Listener
	done   chan struct{} // closed on Close; lets the ctx watcher exit early

	OnDownloadRequested func(BrowserRequest)
	OnError             func(string)
}

// NewBridge builds a bridge on host:port (port 0 = ephemeral, for tests).
func NewBridge(host string, port int) *Bridge {
	cfg := DefaultConfig()
	cfg.Host, cfg.Port = host, port
	return NewBridgeWithConfig(cfg)
}

// NewBridgeWithConfig builds a bridge from an explicit Config.
func NewBridgeWithConfig(cfg Config) *Bridge {
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = MaxBodyBytes
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 2 * time.Second
	}
	return &Bridge{cfg: cfg}
}

// Addr returns the actual bound address (host, port).
func (b *Bridge) Addr() (string, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ln != nil {
		if addr, ok := b.ln.Addr().(*net.TCPAddr); ok {
			return addr.IP.String(), addr.Port
		}
	}
	return b.cfg.Host, b.cfg.Port
}

// Start binds and serves in the background (once). A canceled ctx refuses to
// start; otherwise the bridge lifetime follows ctx — canceling it shuts the
// bridge down with the configured grace.
func (b *Bridge) Start(ctx context.Context) error {
	b.mu.Lock()
	if b.server != nil {
		b.mu.Unlock()
		return nil
	}
	b.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", b.cfg.Host, b.cfg.Port))
	if err != nil {
		return err
	}
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = jsonErrorHandler
	e.Use(middleware.Recover())
	e.Use(corsMiddleware)
	e.Use(middleware.BodyLimit(strconv.FormatInt(b.cfg.MaxBodyBytes, 10)))
	e.GET("/health", b.handleHealth, trustedMiddleware)
	e.POST("/downloads", b.handleDownloads, trustedMiddleware)
	e.OPTIONS("/downloads", b.handleOptions, allowedOrigin)
	srv := &http.Server{Handler: e}
	b.mu.Lock()
	b.ln = ln
	b.server = srv
	b.done = make(chan struct{})
	done := b.done
	b.mu.Unlock()

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			if b.OnError != nil {
				b.OnError(err.Error())
			}
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			timeout, cancel := context.WithTimeout(context.Background(), b.cfg.ShutdownTimeout)
			defer cancel()
			b.Close(timeout)
		case <-done:
			// Closed manually; nothing to do.
		}
	}()

	return nil
}

// Close shuts down gracefully with the configured grace period (a canceled
// ctx still gets the full grace via a fresh background timeout). Take-once
// under lock makes concurrent Close (cleanup vs ctx watcher) safe; repeats
// return nil.
func (b *Bridge) Close(ctx context.Context) error {
	b.mu.Lock()
	srv := b.server
	b.server = nil
	b.ln = nil
	done := b.done
	b.done = nil
	b.mu.Unlock()
	if done != nil {
		close(done)
	}
	if srv == nil {
		return nil
	}
	if ctx.Err() != nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, b.cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(ctx)
}

func (b *Bridge) handleHealth(c echo.Context) error {
	return c.JSON(http.StatusOK, BridgeResponse{"ok": true, "app": "rapid"})
}

func (b *Bridge) handleOptions(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (b *Bridge) handleDownloads(c echo.Context) error {
	var req BrowserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}
	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse(err.Error()))
	}
	if b.OnDownloadRequested != nil {
		b.OnDownloadRequested(req)
	}
	return c.JSON(http.StatusAccepted, BridgeResponse{"ok": true})
}
