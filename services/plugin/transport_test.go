package plugin

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	liberrors "rapid/lib/errors"

	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) Manifest {
	t.Helper()
	m, err := LoadManifest(filepath.Join("testdata", name))
	require.NoError(t, err)
	return m
}

func TestRunResolve(t *testing.T) {
	m := loadFixture(t, "echo")

	items, err := runResolve(t.Context(), m, "https://example.com/watch?v=1", nil)
	require.NoError(t, err)
	require.Equal(t, []ResolvedItem{{
		Title:    "Echo",
		URL:      "https://cdn.example.com/video.mp4",
		Filename: "video.mp4",
		Category: "video",
		MIMEType: "video/mp4",
	}}, items)
}

func TestRunResolvePluginError(t *testing.T) {
	m := loadFixture(t, "error")

	_, err := runResolve(t.Context(), m, "https://error.example/x", nil)
	require.Error(t, err)

	var rpcErr *liberrors.JsonRpcError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, -1, rpcErr.Code)
	require.Contains(t, err.Error(), "boom")
}

func TestRunResolveTimeout(t *testing.T) {
	previous := resolveTimeout
	resolveTimeout = 200 * time.Millisecond
	t.Cleanup(func() { resolveTimeout = previous })

	m := loadFixture(t, "hang")
	start := time.Now()
	_, err := runResolve(t.Context(), m, "https://hang.example/x", nil)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Contains(t, err.Error(), "timeout")
	require.GreaterOrEqual(t, elapsed, 150*time.Millisecond)
	require.Less(t, elapsed, 3*time.Second)
}

func TestPluginCommand(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "plugin.sh")
	require.NoError(t, os.WriteFile(local, nil, 0o644))

	got, err := pluginCommand(Manifest{Dir: dir, Command: "plugin.sh"})
	require.NoError(t, err)
	require.Equal(t, local, got)

	want, err := exec.LookPath("sh")
	require.NoError(t, err)
	got, err = pluginCommand(Manifest{Dir: dir, Command: "sh"})
	require.NoError(t, err)
	require.Equal(t, want, got)

	_, err = pluginCommand(Manifest{Dir: dir, Command: "rapid-definitely-not-a-command"})
	require.Error(t, err)
}

func TestPluginResolver(t *testing.T) {
	m := loadFixture(t, "echo")
	r := NewPluginResolver(m)

	require.True(t, r.ShouldResolve("https://example.com/watch?v=1"))
	require.False(t, r.ShouldResolve("https://other.example/file"))

	items, err := r.Resolve(t.Context(), "https://example.com/watch?v=1", nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Echo", items[0].ResolverName)
	require.Equal(t, "video", items[0].Category.String())
}
