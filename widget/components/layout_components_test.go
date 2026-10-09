package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

func TestHeaderContract(t *testing.T) {
	h := NewHeader()
	require.True(t, h.SearchField != nil && h.MenuButton != nil && h.AddButton != nil, "header controls are incomplete")
	require.Equal(t, "Search...", h.SearchField.Placeholder(), "placeholder")
	require.True(t, h.PreferredSearchWidth() == 350 && h.SearchField.MaximumWidth() == 350 && h.SearchField.MinimumWidth() == 0,
		"search width contract is wrong")
	require.True(t, h.MinimumHeight() >= theme.TouchTarget && h.MenuButton.MinimumHeight() >= theme.TouchTarget && h.AddButton.MinimumHeight() >= theme.TouchTarget,
		"header touch target contract is wrong")
	menus, adds := 0, 0
	h.OnMenuClicked(func() { menus++ })
	h.OnAddClicked(func() { adds++ })
	h.MenuButton.Click()
	h.AddButton.Click()
	require.Equal(t, 1, menus, "menu callback count")
	require.Equal(t, 1, adds, "add callback count")
	h.SetSearchText("abc")
	h.SetPreferredSearchWidth(420)
	require.True(t, h.SearchText() == "abc" && h.PreferredSearchWidth() == 420 && h.SearchField.MaximumWidth() == 420,
		"header state did not round-trip")
}

func TestSidebarItemAndSectionContract(t *testing.T) {
	item := NewSidebarItem("download", "Downloads")
	item.SetCount("3")
	item.SetCategoryItem(true)
	item.SetSelected(true)
	item.SetIconSource(":/icons/MdiLightDownload.svg")
	require.True(t, item.ObjectName() == "sidebarItem" &&
		strings.Contains(item.StyleSheet(), "background: transparent") &&
		strings.Contains(item.StyleSheet(), "border: none") &&
		item.MinimumHeight() == 36 &&
		item.Destination() == "download" &&
		item.Label() == "Downloads" &&
		item.Count() == "3" &&
		item.CountVisible() &&
		item.Selected() &&
		item.iconLabel.Width() == theme.IconXs,
		"sidebar item contract is incomplete")
	selectedBackground := theme.ColorSurface.LighterWithInt(140)
	require.Equal(t, selectedBackground.Name(), item.backgroundColor().Name(), "selected paint color")
	item.SetSelected(false)
	item.hovered = true
	require.Equal(t, selectedBackground.Name(), item.backgroundColor().Name(), "hover paint color")
	activated := 0
	item.OnActivated(func() { activated++ })
	item.activate()
	require.Equal(t, 1, activated, "activation count")

	section := NewSidebarSection()
	section.SetHeading("Navigation")
	section.SetTopMargin(7)
	section.SetItems([]SidebarItemData{
		{Destination: "one", Label: "One"},
		{Destination: "two", Label: "Two", Count: "2"},
	})
	section.SetCurrentDestination("two")
	items := section.ItemWidgets()
	require.True(t, section.HeadingVisible() && section.TopMargin() == 7 && len(items) == 2 && !items[0].Selected() && items[1].Selected(),
		"section model/selection contract is wrong")
	section.SetItems([]SidebarItemData{{Destination: "new", Label: "New"}})
	require.Equal(t, 1, len(section.ItemWidgets()), "section replacement left stale items")
	require.Equal(t, "new", section.ItemWidgets()[0].Destination(), "replacement destination")
}

// A section must re-wire activation for items created by a later SetItems call
// (B21): previously only items present at AddSection time were connected.
func TestSidebarSectionActivationSurvivesSetItems(t *testing.T) {
	section := NewSidebarSection()
	got := ""
	section.OnActivated(func(destination string) { got = destination })

	section.SetItems([]SidebarItemData{{Destination: "one", Label: "One"}})
	section.SetItems([]SidebarItemData{{Destination: "two", Label: "Two"}})
	section.ItemWidgets()[0].activate()
	require.Equal(t, "two", got, "activation after rebuild")
}

func TestSidebarAddSectionForwardsActivation(t *testing.T) {
	sidebar := NewSidebar()
	section := NewSidebarSection()
	section.SetItems([]SidebarItemData{{Destination: "a", Label: "A"}})
	sidebar.AddSection(section)

	// Rebuilt after AddSection: the section wiring must still reach the sidebar.
	section.SetItems([]SidebarItemData{{Destination: "b", Label: "B"}})
	selected := ""
	sidebar.OnDestinationSelected(func(destination string) { selected = destination })
	section.ItemWidgets()[0].activate()

	require.Equal(t, "b", selected, "sidebar did not receive rebuilt item activation")
	require.Equal(t, "b", sidebar.CurrentDestination(), "sidebar current destination")
}

func TestSidebarItemPaintsStateBackground(t *testing.T) {
	item := NewSidebarItem("download", "Downloads")
	item.SetSelected(true)
	window := qt.NewQMainWindow2()
	window.SetCentralWidget(item.QWidget)
	window.Resize(200, 36)
	window.Show()
	qt.QCoreApplication_ProcessEvents()
	selected := item.Grab().ToImage().PixelColor(100, 18)
	want := theme.ColorSurface.LighterWithInt(140)
	require.Equal(t, want.Name(), selected.Name(), "painted selected pixel")
	item.SetSelected(false)
	item.hovered = true
	item.Update()
	qt.QCoreApplication_ProcessEvents()
	hovered := item.Grab().ToImage().PixelColor(100, 18)
	require.Equal(t, want.Name(), hovered.Name(), "painted hover pixel")
	window.Hide()
}

func TestSidebarAndLayoutComposition(t *testing.T) {
	layout := NewLayout()
	section := NewSidebarSection()
	section.SetItems([]SidebarItemData{{Destination: "download", Label: "Downloads", IconSource: ":/icons/MdiLightDownload.svg"}})
	layout.SidebarWidget.AddSection(section)
	layout.SidebarWidget.AddStretch()
	settings := NewSidebarSection()
	settings.SetHeading("Setting")
	settings.SetItems([]SidebarItemData{{Destination: "settings", Label: "Setting", IconSource: ":/icons/MdiLightSettings.svg"}})
	layout.SidebarWidget.AddSection(settings)
	layout.SidebarWidget.SetCurrentDestination("download")
	require.Equal(t, 3, layout.SidebarWidget.ContentLayout.Count(), "sidebar content entries = section/stretch/section")

	window := qt.NewQMainWindow2()
	window.SetCentralWidget(layout.QWidget)
	window.Resize(1024, 700)
	window.Show()
	qt.QCoreApplication_ProcessEvents()
	require.Equal(t, SidebarWidth, layout.SidebarWidget.Width(), "sidebar width")
	require.Greater(t, section.Height(), 0, "section height")
	require.Greater(t, section.ItemWidgets()[0].Height(), 0, "item height")
	require.False(t, section.ItemWidgets()[0].iconLabel.Pixmap2().IsNull(), "sidebar icon did not rasterize")
	require.Equal(t, 1, layout.grid.ColumnStretch(1), "grid column stretch")
	require.Equal(t, 1, layout.grid.RowStretch(1), "grid row stretch")

	before := layout.SidebarOpen()
	layout.HeaderWidget.MenuButton.Click()
	require.NotEqual(t, before, layout.SidebarOpen(), "menu did not toggle sidebar")
	selected := ""
	layout.OnDestinationSelected(func(destination string) { selected = destination })
	layout.SidebarWidget.Activate("settings")
	require.Equal(t, "settings", selected, "destination forwarded after state update")
	require.Equal(t, "settings", layout.SidebarWidget.CurrentDestination(), "sidebar current destination")
	content := qt.NewQLabel3("current testing element")
	layout.AddContentWidget(content.QWidget)
	require.Equal(t, 1, layout.ContentLayout.Count(), "content slot did not receive testing element")
	window.Hide()
}
