package downloads

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/lib"
	"rapid/lib/reactive"
	"rapid/widget/app"
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

func TestMain(m *testing.M) {
	if os.Getenv("QT_QPA_PLATFORM") == "" {
		_ = os.Setenv("QT_QPA_PLATFORM", "offscreen")
	}
	qt.NewQApplication([]string{"rapid-downloads-tests"})
	theme.Init()
	os.Exit(m.Run())
}

func newTestNav() *app.Navigation {
	nav := app.NewNavigation(qt.NewQStackedWidget2())
	nav.RegisterAll(app.RouteMap{
		app.RouteDownload: func(parent *qt.QWidget) *qt.QWidget { return qt.NewQWidget3(parent, 0) },
		app.RouteSettings: func(parent *qt.QWidget) *qt.QWidget { return qt.NewQWidget3(parent, 0) },
	})
	nav.Replace(app.RouteDownload)
	return nav
}

// Clicking "Setting" pushes a page; returning from it must leave the sidebar
// on the destination that was active before the push, not on "settings".
func TestSettingsPushKeepsPreviousSidebarSelection(t *testing.T) {
	nav := newTestNav()
	destination := reactive.NewSignal("all")
	layout := NewLayout(nav)
	layout.BindDestination(destination)

	require.Equal(t, DefaultSidebarItem, destination.Get(), "initial destination")

	layout.SidebarWidget.Activate(lib.CategoryAudio.String())
	require.Equal(t, lib.CategoryAudio.String(), destination.Get(), "after category click")

	layout.SidebarWidget.Activate("settings")
	require.Equal(t, app.RouteSettings, nav.CurrentRoute(), "settings click did not push the settings route")
	require.Equal(t, lib.CategoryAudio.String(), destination.Get(), "settings click stole the selection")

	nav.Back()
	require.Equal(t, app.RouteDownload, nav.CurrentRoute(), "back route")
	require.Equal(t, lib.CategoryAudio.String(), destination.Get(), "destination after back")
}
