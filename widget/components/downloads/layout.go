package downloads

import (
	"rapid/lib"
	"rapid/widget/app"
	"rapid/widget/components"
	"rapid/widget/theme"
	"rapid/widget/ui"
)

const (
	DefaultSidebarItem = "all"
)

// Layout is the components.Layout plus download-specific sidebar counts.
type Layout struct {
	*components.Layout
}

func NewLayout(navigation *app.Navigation) *Layout {
	layout := &Layout{Layout: components.NewLayout()}
	downloads := components.NewSidebarSection()
	downloads.SetItems([]components.SidebarItemData{
		{
			Destination: DefaultSidebarItem,
			Label:       "All downloads",
			IconSource:  ui.IconPath("MdiLightDownload.svg"),
		},
	})
	layout.SidebarWidget.AddSection(downloads)

	categories := components.NewSidebarSection()
	categories.SetTopMargin(theme.SpacingMd)
	categories.SetHeading("CATEGORIES")
	categories.SetItems([]components.SidebarItemData{
		categorySidebarItem(lib.CategoryAudio),
		categorySidebarItem(lib.CategoryApplication),
		categorySidebarItem(lib.CategoryImage),
		categorySidebarItem(lib.CategoryCompressed),
		categorySidebarItem(lib.CategoryDocument),
		categorySidebarItem(lib.CategoryVideo),
		categorySidebarItem(lib.CategoryUnknown),
	})
	layout.SidebarWidget.AddSection(categories)
	layout.SidebarWidget.AddStretch()

	settings := components.NewSidebarSection()
	settings.SetItems([]components.SidebarItemData{
		{
			Label:      "Setting",
			IconSource: ui.IconPath("MdiLightSettings.svg"),
			OnActivated: func() {
				navigation.Push(app.RouteSettings)
			},
		},
	})
	layout.SidebarWidget.AddSection(settings)
	layout.SidebarWidget.SetCurrentDestination(DefaultSidebarItem)

	previous := DefaultSidebarItem
	layout.OnDestinationSelected(func(destination string) {
		if destination != "" {
			previous = destination
			return
		}

		layout.SidebarWidget.SetCurrentDestination(previous)
	})

	return layout
}

// SetCounts updates the sidebar badges, hiding zero counts.
func (l *Layout) SetCounts(counts map[string]int) {
	if l == nil || l.SidebarWidget == nil {
		return
	}
	l.SidebarWidget.SetCounts(counts)
}

func categorySidebarItem(category lib.Category) components.SidebarItemData {
	return components.SidebarItemData{
		Destination:  category.String(),
		Label:        category.Label(),
		IconSource:   ui.IconPath("MdiSquareRounded.svg"),
		IconColor:    theme.CategoryColor(category),
		CategoryItem: true,
	}
}
