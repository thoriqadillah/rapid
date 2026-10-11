package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	base := t.TempDir()
	return Paths{
		AppData:   filepath.Join(base, "appdata"),
		Downloads: filepath.Join(base, "downloads"),
	}
}

func TestDefaultCreatesDirsAndSecret(t *testing.T) {
	p := testPaths(t)
	s, err := Default(t.TempDir(), p)
	require.NoError(t, err)

	require.DirExists(t, p.AppData)
	require.DirExists(t, p.Downloads)
	require.FileExists(t, filepath.Join(p.AppData, "aria2.session"))
	require.NotEmpty(t, s.Aria2Token)
	require.Len(t, s.Aria2Token, 32, "16 random bytes hex-encoded")
	require.Equal(t, p.AppData, s.DataDir)
	require.Equal(t, p.Downloads, s.DownloadDir)
}

func TestDefaultReusesSecret(t *testing.T) {
	p := testPaths(t)
	first, err := Default(t.TempDir(), p)
	require.NoError(t, err)
	second, err := Default(t.TempDir(), p)
	require.NoError(t, err)
	require.Equal(t, first.Aria2Token, second.Aria2Token)
}

func TestDefaultUnwritableDirFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	base := t.TempDir()
	blocked := filepath.Join(base, "blocked")
	require.NoError(t, os.Mkdir(blocked, 0o500))
	_, err := Default(t.TempDir(), Paths{AppData: filepath.Join(blocked, "x"), Downloads: filepath.Join(base, "d")})
	require.Error(t, err)
}

func TestDefaultRejectsEmptyPaths(t *testing.T) {
	_, err := Default(t.TempDir(), Paths{})
	require.Error(t, err)
}

func TestPluginDirsIncludeEnv(t *testing.T) {
	extra := t.TempDir()
	t.Setenv("RAPID_PLUGIN_DIRS", strings.Join([]string{extra, filepath.Join(extra, "two")}, string(os.PathListSeparator)))

	p := testPaths(t)
	base := t.TempDir()
	s, err := Default(base, p)
	require.NoError(t, err)

	require.Equal(t, []string{
		filepath.Join(base, "plugins"),
		filepath.Join(p.AppData, "plugins"),
		extra,
		filepath.Join(extra, "two"),
	}, s.PluginDirs)
}

func TestSystemPathsResolvesBothLocations(t *testing.T) {
	p, err := SystemPaths()
	require.NoError(t, err)
	require.NotEmpty(t, p.AppData)
	require.Contains(t, p.AppData, "rapid")
	require.NotEmpty(t, p.Downloads)
}

func TestXDGDownloadDir(t *testing.T) {
	home := t.TempDir()
	cfg := filepath.Join(home, ".config")
	require.NoError(t, os.MkdirAll(cfg, 0o755))

	require.Empty(t, xdgDownloadDir(home), "missing file")

	err := os.WriteFile(filepath.Join(cfg, "user-dirs.dirs"), []byte("# comment\nXDG_DOWNLOAD_DIR=\"$HOME/Dlname\"\n"), 0o644)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "Dlname"), xdgDownloadDir(home))
}
