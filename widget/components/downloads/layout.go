package downloads

import (
	"rapid/lib"
	"rapid/widget/components"
	"rapid/widget/theme"
	"rapid/widget/ui"
)

func NewLayout() *components.Layout {
	layout := components.NewLayout()
	downloads := components.NewSidebarSection()
	downloads.SetItems([]components.SidebarItemData{
		{Destination: "all", Label: "All downloads", IconSource: ui.IconPath("MdiLightDownload.svg")},
	})
	layout.SidebarWidget.AddSection(downloads)

	categories := components.NewSidebarSection()
	categories.SetTopMargin(theme.SpacingMd)
	categories.SetHeading("CATEGORIES")
	categories.SetItems([]components.SidebarItemData{
		{Destination: lib.Audio.String(), Label: lib.Audio.Label(), IconSource: ui.IconPath("MdiSquareRounded.svg"), IconColor: theme.CategoryColor(lib.Audio), CategoryItem: true},
		{Destination: lib.Application.String(), Label: lib.Application.Label(), IconSource: ui.IconPath("MdiSquareRounded.svg"), IconColor: theme.CategoryColor(lib.Application), CategoryItem: true},
		{Destination: lib.Image.String(), Label: lib.Image.Label(), IconSource: ui.IconPath("MdiSquareRounded.svg"), IconColor: theme.CategoryColor(lib.Image), CategoryItem: true},
		{Destination: lib.Compressed.String(), Label: lib.Compressed.Label(), IconSource: ui.IconPath("MdiSquareRounded.svg"), IconColor: theme.CategoryColor(lib.Compressed), CategoryItem: true},
		{Destination: lib.Document.String(), Label: lib.Document.Label(), IconSource: ui.IconPath("MdiSquareRounded.svg"), IconColor: theme.CategoryColor(lib.Document), CategoryItem: true},
		{Destination: lib.Video.String(), Label: lib.Video.Label(), IconSource: ui.IconPath("MdiSquareRounded.svg"), IconColor: theme.CategoryColor(lib.Video), CategoryItem: true},
		{Destination: lib.Unknown.String(), Label: lib.Unknown.Label(), IconSource: ui.IconPath("MdiSquareRounded.svg"), IconColor: theme.CategoryColor(lib.Unknown), CategoryItem: true},
	})
	layout.SidebarWidget.AddSection(categories)
	layout.SidebarWidget.AddStretch()

	settings := components.NewSidebarSection()
	settings.SetItems([]components.SidebarItemData{
		{Destination: "settings", Label: "Setting", IconSource: ui.IconPath("MdiLightSettings.svg")},
	})
	layout.SidebarWidget.AddSection(settings)
	layout.SidebarWidget.SetCurrentDestination("all")

	return layout
}
