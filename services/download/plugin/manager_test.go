package plugin

import (
	"os"
	"path/filepath"
	"testing"

	liberrors "rapid/lib/errors"

	"github.com/stretchr/testify/require"
)

func writePluginDir(t *testing.T, root, name, manifest string) {
	t.Helper()
	dir := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ManifestName), []byte(manifest), 0o644))
}

func copyFixture(t *testing.T, root, name, fixture string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", fixture, ManifestName))
	require.NoError(t, err)
	writePluginDir(t, root, name, string(body))
}

func TestManagerSkipsBroken(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, root, "valid", "echo")
	writePluginDir(t, root, "broken", `{not json`)

	m := NewManager([]string{root})
	require.Equal(t, 1, m.Len())

	items, err := m.Resolve(t.Context(), "https://example.com/watch", nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Echo", items[0].ResolverName)
}

func TestManagerFirstDirWins(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	writePluginDir(t, first, "p", `{"id":"dupe","name":"First","type":"resolver","match":["x"],"transport":{"command":"printf"}}`)
	writePluginDir(t, second, "p", `{"id":"dupe","name":"Second","type":"resolver","match":["x"],"transport":{"command":"printf"}}`)

	m := NewManager([]string{first, second})
	require.Equal(t, 1, m.Len())
	require.Equal(t, "First", m.plugins[0].manifest.Name)
}

func TestManagerNoMatch(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, root, "valid", "echo")

	m := NewManager([]string{root})
	items, err := m.Resolve(t.Context(), "https://other.example/file", nil)
	require.NoError(t, err)
	require.Empty(t, items)
}

func TestManagerPluginFailure(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, root, "failing", "error")

	m := NewManager([]string{root})
	_, err := m.Resolve(t.Context(), "https://error.example/x", nil)
	require.Error(t, err)

	var rpcErr *liberrors.JsonRpcError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, -1, rpcErr.Code)
}
