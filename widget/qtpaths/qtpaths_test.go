package qtpaths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	qt "github.com/mappu/miqt/qt6"

	"rapid/services/settings"
)

func TestMain(m *testing.M) {
	if os.Getenv("QT_QPA_PLATFORM") == "" {
		_ = os.Setenv("QT_QPA_PLATFORM", "offscreen")
	}
	qt.NewQApplication([]string{"rapid-qtpaths-tests"})
	qt.QCoreApplication_SetApplicationName("rapid")
	os.Exit(m.Run())
}

// qtpaths.Resolve must agree with settings.SystemPaths so headless tools
// (cmd/seed) and the app open the same database (Linux only: Qt and the XDG
// spec agree there; other platforms resolve differently by design).
func TestResolveMatchesSystemPaths(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Qt/XDG location agreement is only guaranteed on Linux")
	}
	if x := os.Getenv("XDG_DATA_HOME"); x == "" {
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	}

	got := Resolve()
	want, err := settings.SystemPaths()
	require.NoError(t, err)
	require.Equal(t, want.AppData, got.AppData)
	require.NotEmpty(t, got.Downloads)
}
