package app

import (
	"os"
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if os.Getenv("QT_QPA_PLATFORM") == "" {
		_ = os.Setenv("QT_QPA_PLATFORM", "offscreen")
	}
	qt.NewQApplication([]string{"rapid-nav-tests"})
	os.Exit(m.Run())
}

func testRoutes() RouteMap {
	return RouteMap{
		"download": func(parent *qt.QWidget) *qt.QWidget { return qt.NewQWidget3(parent, 0) },
		"settings": func(parent *qt.QWidget) *qt.QWidget { return qt.NewQWidget3(parent, 0) },
	}
}

func TestNavigationBackReturnsToPreviousRoute(t *testing.T) {
	nav := NewNavigation(qt.NewQStackedWidget2())
	nav.RegisterAll(testRoutes())

	require.True(t, nav.Replace("download"), "Replace(download) failed")
	require.Equal(t, 1, nav.Depth(), "after replace depth")
	require.Equal(t, "download", nav.CurrentRoute(), "after replace route")

	require.True(t, nav.Push("settings"), "Push(settings) failed")
	require.Equal(t, 2, nav.Depth(), "after push depth")
	require.Equal(t, "settings", nav.CurrentRoute(), "after push route")

	// Settings' "Back" item calls Back(). This must pop one page, not replace
	// the top entry (which previously left [download, download]).
	require.True(t, nav.Back(), "Back() failed")
	require.Equal(t, 1, nav.Depth(), "after back depth")
	require.Equal(t, "download", nav.CurrentRoute(), "after back route")

	// Popping past the root is a no-op.
	require.False(t, nav.Back(), "Back() on the root page should be a no-op")
	require.Equal(t, 1, nav.Depth(), "depth changed on root Back")
}

func TestNavigationPushBackDoesNotLeakWidgets(t *testing.T) {
	nav := NewNavigation(qt.NewQStackedWidget2())
	nav.RegisterAll(testRoutes())
	nav.Replace("download")

	drainDeferredDeletes := func() {
		qt.QCoreApplication_SendPostedEvents2(nil, int(qt.QEvent__DeferredDelete))
	}
	drainDeferredDeletes()
	base := len(qt.QApplication_AllWidgets())

	for range 100 {
		nav.Push("settings")
		nav.Back()
		drainDeferredDeletes()
	}

	got := len(qt.QApplication_AllWidgets())
	require.LessOrEqual(t, got, base+5, "pushed pages leaked: widget count %d -> %d", base, got)
}
