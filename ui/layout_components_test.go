package ui

import (
	"strings"
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"rapid/theme"
)

func TestHeaderContract(t *testing.T) {
	h := NewHeader()
	if h.SearchField == nil || h.MenuButton == nil || h.AddButton == nil {
		t.Fatal("header controls are incomplete")
	}
	if h.SearchField.Placeholder() != "Search..." {
		t.Fatalf("placeholder = %q", h.SearchField.Placeholder())
	}
	if h.PreferredSearchWidth() != 350 || h.SearchField.MaximumWidth() != 350 || h.SearchField.MinimumWidth() != 0 {
		t.Fatal("search width contract is wrong")
	}
	if h.MinimumHeight() < theme.TouchTarget || h.MenuButton.MinimumHeight() < theme.TouchTarget || h.AddButton.MinimumHeight() < theme.TouchTarget {
		t.Fatal("header touch target contract is wrong")
	}
	menus, adds := 0, 0
	h.OnMenuClicked(func() { menus++ })
	h.OnAddClicked(func() { adds++ })
	h.MenuButton.Click()
	h.AddButton.Click()
	if menus != 1 || adds != 1 {
		t.Fatalf("callbacks menu=%d add=%d", menus, adds)
	}
	h.SetSearchText("abc")
	h.SetPreferredSearchWidth(420)
	if h.SearchText() != "abc" || h.PreferredSearchWidth() != 420 || h.SearchField.MaximumWidth() != 420 {
		t.Fatal("header state did not round-trip")
	}
}

func TestSidebarItemAndSectionContract(t *testing.T) {
	item := NewSidebarItem("download", "Downloads")
	item.SetCount("3")
	item.SetCategoryItem(true)
	item.SetSelected(true)
	item.SetIconSource(":/icons/MdiLightDownload.svg")
	if item.ObjectName() != "sidebarItem" || !strings.Contains(item.StyleSheet(), "background: transparent") || !strings.Contains(item.StyleSheet(), "border: none") || item.MinimumHeight() != 36 || item.Destination() != "download" || item.Label() != "Downloads" || item.Count() != "3" || !item.CountVisible() || !item.Selected() || item.iconLabel.Width() != theme.IconXs {
		t.Fatal("sidebar item contract is incomplete")
	}
	selectedBackground := theme.ColorSurface.LighterWithInt(140)
	if item.backgroundColor().Name() != selectedBackground.Name() {
		t.Fatalf("selected paint color = %s, want %s", item.backgroundColor().Name(), selectedBackground.Name())
	}
	item.SetSelected(false)
	item.hovered = true
	if item.backgroundColor().Name() != selectedBackground.Name() {
		t.Fatalf("hover paint color = %s, want %s", item.backgroundColor().Name(), selectedBackground.Name())
	}
	activated := 0
	item.OnActivated(func() { activated++ })
	item.activate()
	if activated != 1 {
		t.Fatalf("activation count = %d", activated)
	}

	section := NewSidebarSection()
	section.SetHeading("Navigation")
	section.SetTopMargin(7)
	section.SetItems([]SidebarItemData{
		{Destination: "one", Label: "One"},
		{Destination: "two", Label: "Two", Count: "2"},
	})
	section.SetCurrentDestination("two")
	items := section.ItemWidgets()
	if !section.HeadingVisible() || section.TopMargin() != 7 || len(items) != 2 || items[0].Selected() || !items[1].Selected() {
		t.Fatal("section model/selection contract is wrong")
	}
	seen := ""
	section.OnActivated(func(destination string) { seen = destination })
	items[0].activate()
	if seen != "one" {
		t.Fatalf("section destination = %q", seen)
	}
	section.SetItems([]SidebarItemData{{Destination: "new", Label: "New"}})
	if len(section.ItemWidgets()) != 1 || section.ItemWidgets()[0].Destination() != "new" {
		t.Fatal("section replacement left stale items")
	}
}

func TestSidebarItemPaintsStateBackground(t *testing.T) {
	item := NewSidebarItem("download", "Downloads")
	item.SetSelected(true)
	window := qt.NewQMainWindow2()
	window.SetCentralWidget(item.QWidget)
	window.Resize(200, 36)
	window.Show()
	processEvents()
	selected := item.Grab().ToImage().PixelColor(100, 18)
	want := theme.ColorSurface.LighterWithInt(140)
	if selected.Name() != want.Name() {
		t.Fatalf("painted selected pixel = %s, want %s", selected.Name(), want.Name())
	}
	item.SetSelected(false)
	item.hovered = true
	item.Update()
	processEvents()
	hovered := item.Grab().ToImage().PixelColor(100, 18)
	if hovered.Name() != want.Name() {
		t.Fatalf("painted hover pixel = %s, want %s", hovered.Name(), want.Name())
	}
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
	if layout.SidebarWidget.ContentLayout.Count() != 3 {
		t.Fatalf("sidebar content entries = %d, want section/stretch/section", layout.SidebarWidget.ContentLayout.Count())
	}

	window := qt.NewQMainWindow2()
	window.SetCentralWidget(layout.QWidget)
	window.Resize(1024, 700)
	window.Show()
	processEvents()
	if layout.SidebarWidget.Width() != SidebarWidth || section.Height() <= 0 || section.ItemWidgets()[0].Height() <= 0 {
		t.Fatalf("unexpected geometry sidebar=%d section=%d item=%d", layout.SidebarWidget.Width(), section.Height(), section.ItemWidgets()[0].Height())
	}
	if section.ItemWidgets()[0].iconLabel.Pixmap2().IsNull() {
		t.Fatal("sidebar icon did not rasterize")
	}
	if layout.grid.ColumnStretch(1) != 1 || layout.grid.RowStretch(1) != 1 {
		t.Fatal("content grid does not stretch")
	}

	before := layout.SidebarOpen()
	layout.HeaderWidget.MenuButton.Click()
	if layout.SidebarOpen() == before {
		t.Fatal("menu did not toggle sidebar")
	}
	selected := ""
	layout.OnDestinationSelected(func(destination string) { selected = destination })
	layout.SidebarWidget.Activate("settings")
	if selected != "settings" || layout.SidebarWidget.CurrentDestination() != "settings" {
		t.Fatal("destination was not forwarded after state update")
	}
	content := qt.NewQLabel3("current testing element")
	layout.AddContentWidget(content.QWidget)
	if layout.ContentLayout.Count() != 1 {
		t.Fatal("content slot did not receive testing element")
	}
	window.Hide()
}
