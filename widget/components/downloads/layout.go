package downloads

import (
	"rapid/lib"
	"rapid/lib/reactive"
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
	layout.SidebarWidget.SetActive(DefaultSidebarItem)

	return layout
}

// BindCounts updates the sidebar badges, hiding zero counts.
func (l *Layout) BindCounts(counts *reactive.Computed[map[string]int]) {
	if l.SidebarWidget == nil {
		return
	}

	l.OnDestroyed(reactive.Effect(func() {
		l.SidebarWidget.SetCounts(counts.Get())
	}))
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
