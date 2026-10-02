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

func NewLayout(navigation *app.Navigation) *components.Layout {
	layout := components.NewLayout()
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
		{
			Destination:  lib.CategoryAudio.String(),
			Label:        lib.CategoryAudio.Label(),
			IconSource:   ui.IconPath("MdiSquareRounded.svg"),
			IconColor:    theme.CategoryColor(lib.CategoryAudio),
			CategoryItem: true,
		},
		{
			Destination:  lib.CategoryApplication.String(),
			Label:        lib.CategoryApplication.Label(),
			IconSource:   ui.IconPath("MdiSquareRounded.svg"),
			IconColor:    theme.CategoryColor(lib.CategoryApplication),
			CategoryItem: true,
		},
		{
			Destination:  lib.CategoryImage.String(),
			Label:        lib.CategoryImage.Label(),
			IconSource:   ui.IconPath("MdiSquareRounded.svg"),
			IconColor:    theme.CategoryColor(lib.CategoryImage),
			CategoryItem: true,
		},
		{
			Destination:  lib.CategoryCompressed.String(),
			Label:        lib.CategoryCompressed.Label(),
			IconSource:   ui.IconPath("MdiSquareRounded.svg"),
			IconColor:    theme.CategoryColor(lib.CategoryCompressed),
			CategoryItem: true,
		},
		{
			Destination:  lib.CategoryDocument.String(),
			Label:        lib.CategoryDocument.Label(),
			IconSource:   ui.IconPath("MdiSquareRounded.svg"),
			IconColor:    theme.CategoryColor(lib.CategoryDocument),
			CategoryItem: true,
		},
		{
			Destination:  lib.CategoryVideo.String(),
			Label:        lib.CategoryVideo.Label(),
			IconSource:   ui.IconPath("MdiSquareRounded.svg"),
			IconColor:    theme.CategoryColor(lib.CategoryVideo),
			CategoryItem: true,
		},
		{
			Destination:  lib.CategoryUnknown.String(),
			Label:        lib.CategoryUnknown.Label(),
			IconSource:   ui.IconPath("MdiSquareRounded.svg"),
			IconColor:    theme.CategoryColor(lib.CategoryUnknown),
			CategoryItem: true,
		},
	})
	layout.SidebarWidget.AddSection(categories)
	layout.SidebarWidget.AddStretch()

	settings := components.NewSidebarSection()
	settings.SetItems([]components.SidebarItemData{
		{
			Destination: "settings",
			Label:       "Setting",
			IconSource:  ui.IconPath("MdiLightSettings.svg"),
			OnActivated: func() {
				navigation.Push(app.RouteSettings)
			},
		},
	})
	layout.SidebarWidget.AddSection(settings)
	layout.SidebarWidget.SetCurrentDestination(DefaultSidebarItem)

	return layout
}
