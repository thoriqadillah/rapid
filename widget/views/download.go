package views

import (
	"fmt"
	"rapid/service/notification"
	"rapid/widget/app"
	"rapid/widget/components/downloads"
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

func NewDownloadView(parent *qt.QWidget, notifier *notification.Service, navigation *app.Navigation) *qt.QWidget {
	layout := downloads.NewLayout(navigation)

	content := qt.NewQWidget2()
	contentLayout := qt.NewQVBoxLayout2()
	contentLayout.SetContentsMargins(theme.SpacingXl, theme.SpacingXl, theme.SpacingXl, theme.SpacingXl)
	contentLayout.SetSpacing(theme.SpacingLg)
	content.SetLayout(contentLayout.QLayout)

	panel := qt.NewQWidget2()
	panel.SetMaximumWidth(820)
	panel.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Preferred)
	panel.SetStyleSheet(fmt.Sprintf(
		"QWidget { background-color: %s; border-radius: %dpx; }",
		theme.CssColor(theme.ColorSurface), theme.RadiusMd,
	))
	panelLayout := qt.NewQVBoxLayout2()
	panelLayout.SetContentsMargins(theme.SpacingXl, theme.SpacingXl, theme.SpacingXl, theme.SpacingXl)
	panelLayout.SetSpacing(theme.SpacingLg)
	panel.SetLayout(panelLayout.QLayout)

	title := qt.NewQLabel3("component demo")
	title.SetStyleSheet("font-size: 20px; color: " + theme.CssColor(theme.ColorText) + "; background: transparent;")
	panelLayout.AddWidget(title.QWidget)

	routeStatus := qt.NewQLabel3("Selected: download-1")
	routeStatus.SetStyleSheet("color: " + theme.CssColor(theme.ColorTextMuted) + "; background: transparent;")
	panelLayout.AddWidget(routeStatus.QWidget)

	buttons := qt.NewQHBoxLayout2()
	buttons.SetSpacing(theme.SpacingSm)
	panelLayout.AddLayout(buttons.QLayout)
	buttons.AddWidget(ui.NewRButton("Base", ui.BaseVariant, false).QWidget)
	buttons.AddWidget(ui.NewRButton("Primary", ui.PrimaryVariant, false).QWidget)
	buttons.AddWidget(ui.NewRButton("Danger", ui.DangerVariant, false).QWidget)
	buttons.AddWidget(ui.NewRButton("Link", ui.LinkVariant, false).QWidget)
	buttons.AddWidget(ui.NewRButton("Outline", ui.PrimaryVariant, true).QWidget)
	buttons.AddWidget(ui.NewRButtonIcon(ui.IconPath("MdiLightPlus.svg"), ui.BaseVariant, false).QWidget)

	iconButton := ui.NewRButton("New", ui.PrimaryVariant, false)
	iconButton.SetIconSource(ui.IconPath("MdiLightPlus.svg"))
	buttons.AddWidget(iconButton.QWidget)

	disabledButton := ui.NewRButton("Disabled", ui.PrimaryVariant, false)
	disabledButton.SetDisabled(true)
	buttons.AddWidget(disabledButton.QWidget)

	panelLayout.AddWidget(ui.NewRSwitch(true).QWidget)

	field := ui.NewRTextField()
	field.SetLabel("URL")
	field.SetPlaceholder("https://example.com")
	field.SetError("invalid URL")
	field.SetPrefixIcon(ui.IconPath("MdiLightContentPaste.svg"))
	panelLayout.AddWidget(field.QWidget)

	downloadDialog := downloads.NewDownloadDialog(parent)

	dialogButton := ui.NewRButton("Open dialog", ui.SecondaryVariant, false)
	dialogButton.OnClicked(func() {
		downloadDialog.Open()
	})
	panelLayout.AddWidget(dialogButton.QWidget)

	notificationButton := ui.NewRButton("Open notification", ui.SecondaryVariant, false)
	notificationButton.OnClicked(func() {
		notifier.Info("Hello", "World", true)
	})
	panelLayout.AddWidget(notificationButton.QWidget)
	contentLayout.AddStretch()
	contentLayout.AddWidget(panel)
	contentLayout.SetAlignment(panel, qt.AlignHCenter)
	contentLayout.AddStretch()
	layout.AddContentWidget(content)

	layout.OnDestinationSelected(func(destination string) {
		routeStatus.SetText("Selected: " + destination)
	})
	layout.OnAddClicked(func() {
		downloadDialog.Open()
	})
	layout.HeaderWidget.SearchField.OnTextChanged(func(text string) {
		if text == "" {
			routeStatus.SetText("Selected: " + layout.SidebarWidget.CurrentDestination())
			return
		}
		routeStatus.SetText("Search: " + text)
	})

	return layout.QWidget
}
