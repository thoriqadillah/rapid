package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decode(t *testing.T, payload map[string]any) BrowserRequest {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	var req BrowserRequest
	require.NoError(t, json.Unmarshal(body, &req))
	require.NoError(t, req.Validate())
	return req
}

func TestNormalizePreservesDownloadContext(t *testing.T) {
	req := decode(t, map[string]any{
		"url":     "https://cdn.example/video.mp4",
		"pageUrl": "https://example/watch/1",
		"referer": "https://example/watch/1",
		"title":   "Example video",
		"headers": map[string]any{
			"Authorization":  "Bearer token",
			"Origin":         "https://example",
			"Host":           "forged.example",
			"Content-Length": "42",
			"Count":          7,
		},
		"cookies": map[string]any{
			"session": "secret",
		},
	})
	assert.Equal(t, "https://cdn.example/video.mp4", req.URL)
	assert.Equal(t, "https://example/watch/1", req.PageURL)
	assert.Equal(t, "https://example/watch/1", req.Referer)
	assert.Equal(t, "Example video", req.Title)
	assert.Equal(t, "Bearer token", req.Headers["Authorization"])
	assert.Equal(t, "https://example", req.Headers["Origin"])
	assert.NotContains(t, req.Headers, "Host", "Host must be stripped")
	assert.NotContains(t, req.Headers, "Content-Length", "Content-Length must be stripped")
	// Non-strings pass through the bridge untouched (JSON numbers decode as
	// float64); the service drops them via strMapOf before aria2 sees them.
	assert.Equal(t, float64(7), req.Headers["Count"])
	assert.Equal(t, "secret", req.Cookies["session"])
}

func TestNormalizeAcceptsPreresolvedMetadata(t *testing.T) {
	req := decode(t, map[string]any{
		"url":             "https://example.com/video.mp4",
		"browserResolved": true,
		"size":            2048,
		"category":        "video",
		"bogus":           1,
	})
	assert.True(t, req.BrowserResolved)
	require.NotNil(t, req.Size)
	assert.Equal(t, int64(2048), *req.Size)
	assert.Equal(t, "video", req.Category)
}

func TestNormalizeRejectsNonDownloadable(t *testing.T) {
	var req BrowserRequest
	require.NoError(t, json.Unmarshal([]byte(`{"url": "blob:https://example/id"}`), &req))
	err := req.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "downloadable")

	var garbage BrowserRequest
	err = json.Unmarshal([]byte(`"nope"`), &garbage)
	require.Error(t, err)
	err = json.Unmarshal([]byte(`not json`), &garbage)
	require.Error(t, err)
}

func TestNormalizeDropsNegativeSize(t *testing.T) {
	req := decode(t, map[string]any{
		"url":  "https://example.com/f",
		"size": -1,
	})
	assert.Nil(t, req.Size)
}

func startTestBridge(t *testing.T) (*Bridge, string) {
	t.Helper()
	b := NewBridge("127.0.0.1", 0)
	require.NoError(t, b.Start(t.Context()))
	t.Cleanup(func() { b.Close(t.Context()) })
	host, port := b.Addr()
	return b, fmt.Sprintf("http://%s:%d", host, port)
}

func TestBridgePreflightAndDispatch(t *testing.T) {
	for _, origin := range []string{"chrome-extension://id", "moz-extension://id"} {
		b, base := startTestBridge(t)
		var received []BrowserRequest
		b.OnDownloadRequested = func(r BrowserRequest) {
			received = append(received, r)
		}

		pre, err := http.NewRequest(http.MethodOptions, base+"/downloads", nil)
		require.NoError(t, err)
		pre.Header.Set("Origin", origin)
		pre.Header.Set("Access-Control-Request-Headers", "content-type, x-rapid-extension")
		pre.Header.Set("Access-Control-Request-Method", "POST")
		resp, err := http.DefaultClient.Do(pre)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, 204, resp.StatusCode, "preflight")
		assert.Equal(t, origin, resp.Header.Get("Access-Control-Allow-Origin"), "CORS echo")

		body, _ := json.Marshal(map[string]any{"url": "https://example.com/video.mp4"})
		post, _ := http.NewRequest(http.MethodPost, base+"/downloads", bytes.NewReader(body))
		post.Header.Set("Origin", origin)
		post.Header.Set("Content-Type", "application/json")
		post.Header.Set("X-Rapid-Extension", "1")
		resp, err = http.DefaultClient.Do(post)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, 202, resp.StatusCode, "post")
		require.Len(t, received, 1)
		assert.Equal(t, "https://example.com/video.mp4", received[0].URL)
	}
}

func TestBridgeRejectsUntrusted(t *testing.T) {
	_, base := startTestBridge(t)
	body, _ := json.Marshal(map[string]any{
		"url": "https://example.com/video.mp4",
	})

	// No extension header.
	resp, err := http.Post(base+"/downloads", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, 403, resp.StatusCode)

	// Bad origin preflight.
	pre, _ := http.NewRequest(http.MethodOptions, base+"/downloads", nil)
	pre.Header.Set("Origin", "https://evil.example")
	pre.Header.Set("Access-Control-Request-Headers", "x-rapid-extension")
	resp, err = http.DefaultClient.Do(pre)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, 403, resp.StatusCode)

	// Oversized body.
	big := bytes.Repeat([]byte("a"), MaxBodyBytes+1)
	req, _ := http.NewRequest(http.MethodPost, base+"/downloads", bytes.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rapid-Extension", "1")
	req.ContentLength = int64(len(big))
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, 413, resp.StatusCode)
}

func TestBridgeHealth(t *testing.T) {
	_, base := startTestBridge(t)
	req, _ := http.NewRequest(http.MethodGet, base+"/health", nil)
	req.Header.Set("X-Rapid-Extension", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)
	// Untrusted health -> 403 like every other trusted route (uniform trust
	// mechanism; deliberate divergence from Python's hide-existence 404).
	resp, err = http.Get(base + "/health")
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, 403, resp.StatusCode)
}

func TestBridgeStartRefusesCanceledCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	b := NewBridge("127.0.0.1", 0)
	assert.ErrorIs(t, b.Start(ctx), context.Canceled)
}

func TestBridgeErrorShape(t *testing.T) {
	_, base := startTestBridge(t)
	// Malformed JSON -> 400 in contract shape.
	req, _ := http.NewRequest(http.MethodPost, base+"/downloads", bytes.NewReader([]byte("nope{")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rapid-Extension", "1")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	_ = resp.Body.Close()
	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, false, payload["ok"])
	assert.NotEmpty(t, payload["error"])

	// Wrong method -> 405 in contract shape (Echo router, not the handler).
	put, _ := http.NewRequest(http.MethodPut, base+"/downloads", nil)
	put.Header.Set("X-Rapid-Extension", "1")
	resp, err = http.DefaultClient.Do(put)
	require.NoError(t, err)
	payload = nil
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	_ = resp.Body.Close()
	assert.Equal(t, 405, resp.StatusCode)
	assert.Equal(t, false, payload["ok"])
}

func TestBridgeCustomConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Port = 0
	cfg.MaxBodyBytes = 64
	b := NewBridgeWithConfig(cfg)
	require.NoError(t, b.Start(t.Context()))
	t.Cleanup(func() { _ = b.Close(t.Context()) })
	host, port := b.Addr()
	base := fmt.Sprintf("http://%s:%d", host, port)

	big := bytes.Repeat([]byte("a"), 128)
	req, _ := http.NewRequest(http.MethodPost, base+"/downloads", bytes.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rapid-Extension", "1")
	req.ContentLength = int64(len(big))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, 413, resp.StatusCode)
}

func TestBridgeFollowsCtxLifetime(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	b := NewBridge("127.0.0.1", 0)
	require.NoError(t, b.Start(ctx))
	host, port := b.Addr()
	assert.NotZero(t, port)
	cancel()
	require.Eventually(t, func() bool {
		c, err := http.Get(fmt.Sprintf("http://%s:%d/health", host, port))
		if err != nil {
			return true
		}
		_ = c.Body.Close()
		return false
	}, 5*time.Second, 50*time.Millisecond, "bridge must shut down with ctx")
}
