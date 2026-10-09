package rpc

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"rapid/services/download/rpc/api"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRPCMapsTransportFailure(t *testing.T) {
	rpc := NewJSONRpc("127.0.0.1", 1, "") // closed port
	_, err := rpc.Call(t.Context(), "aria2.getVersion", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot connect to aria2")
}

func TestRPCIncrementsID(t *testing.T) {
	var ids []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		ids = append(ids, int(payload["id"].(float64)))
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": payload["id"], "result": "ok"})
	}))
	defer srv.Close()
	rpc := NewJSONRpc("", 0, "")
	rpc.BaseURL = srv.URL
	_, err := rpc.Call(t.Context(), "m", nil)
	require.NoError(t, err)
	_, err = rpc.Call(t.Context(), "m", nil)
	require.NoError(t, err)
	require.Len(t, ids, 2)
	require.Equal(t, ids[0]+1, ids[1])
}

func TestRPCReturnsRawBody(t *testing.T) {
	// Success bodies come back untouched (envelope included); each
	// consumer strips what it needs.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result": map[string]any{
				"version": "1.0",
			},
		})
	}))
	defer srv.Close()

	rpc := NewJSONRpc("", 0, "")
	rpc.BaseURL = srv.URL
	raw, err := rpc.Call(t.Context(), "aria2.getVersion", nil)
	require.NoError(t, err)
	var env struct {
		Result struct {
			Version string `json:"version"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &env))
	require.Equal(t, "1.0", env.Result.Version)
}

func spawnDaemon(t *testing.T, secret string) int {
	t.Helper()
	if _, err := exec.LookPath("aria2c"); err != nil {
		t.Skip("aria2c not installed")
	}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	dir := t.TempDir()
	session := filepath.Join(dir, "s.session")
	require.NoError(t, os.WriteFile(session, nil, 0o644))
	args := []string{
		"--enable-rpc",
		"--rpc-listen-port=" + strconv.Itoa(port),
		"--dir=" + dir,
		"--save-session=" + session,
		"--input-file=" + session,
	}
	if secret != "" {
		args = append(args, "--rpc-secret="+secret)
	}
	cmd := exec.Command("aria2c", args...)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		c, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err == nil {
			_ = c.Close()
			return port
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.FailNow(t, "aria2 RPC port never opened")
	return 0
}

func TestIntegrationGetVersion(t *testing.T) {
	port := spawnDaemon(t, "")
	rpc := NewJSONRpc("127.0.0.1", port, "")
	raw, err := rpc.Call(t.Context(), "aria2.getVersion", nil)
	require.NoError(t, err)
	var v api.Ari2VersionResponse
	require.NoError(t, json.Unmarshal(raw, &v))
	require.NotEmpty(t, v.Result.Version)
}

func TestIntegrationTellStatusError(t *testing.T) {
	port := spawnDaemon(t, "")
	rpc := NewJSONRpc("127.0.0.1", port, "")
	b, err := json.Marshal([]any{"dead-gid"})
	require.NoError(t, err)
	_, err = rpc.Call(t.Context(), "aria2.tellStatus", b)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid GID")
}

func TestIntegrationTokenAuth(t *testing.T) {
	port := spawnDaemon(t, "s3cr3t")
	good := NewJSONRpc("127.0.0.1", port, "s3cr3t")
	_, err := good.Call(t.Context(), "aria2.getVersion", nil)
	require.NoError(t, err)
	bad := NewJSONRpc("127.0.0.1", port, "")
	_, err = bad.Call(t.Context(), "aria2.getVersion", nil)
	require.Error(t, err)
}
