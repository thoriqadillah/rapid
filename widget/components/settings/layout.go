package settings

import (
	"rapid/widget/app"
	"rapid/widget/components"
	"rapid/widget/ui"
)

func NewLayout(navigation *app.Navigation) *components.Layout {
	layout := components.NewLayout()
	general := components.NewSidebarSection()
	general.SetItems([]components.SidebarItemData{
		{
			Destination: "general",
			Label:       "General",
			IconSource:  ui.IconPath("MdiLightDownload.svg"),
		},
		{
			Destination: "personalization",
			Label:       "Personalization",
			IconSource:  ui.IconPath("MdiLightAccount.svg"),
		},
	})
	layout.SidebarWidget.AddSection(general)
	layout.SidebarWidget.AddStretch()

	downloads := components.NewSidebarSection()
	downloads.SetItems([]components.SidebarItemData{
		{
			Destination: "back",
			Label:       "Back",
			IconSource:  ui.IconPath("MdiLightArrowLeft.svg"),
			OnActivated: func() {
				navigation.Back()
			},
		},
	})
	layout.SidebarWidget.AddSection(downloads)
	layout.SidebarWidget.Activate("general")

	return layout
}
