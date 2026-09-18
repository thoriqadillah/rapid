package main

import (
	"fmt"
	"os"

	"rapid/lib"
	"rapid/widget/components"
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

func main() {
	qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetApplicationName("Rapid")
	theme.Init()

	win := qt.NewQMainWindow2()
	win.SetWindowTitle("Rapid")
	win.Resize(1024, 700)
	win.SetStyleSheet(fmt.Sprintf(`
		QMainWindow { color: %s; font-size: %dpx; }
	`, theme.CssColor(theme.ColorText), theme.TextSize))

	layout := components.NewLayout()
	win.SetCentralWidget(layout.QWidget)

	downloads := components.NewSidebarSection()
	downloads.SetHeading("LIBRARY")
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

	mainContent(win, layout)

	win.Show()
	qt.QApplication_Exec()
}

func mainContent(win *qt.QMainWindow, layout *components.Layout) {
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

	openDialog := func() {
		dialog := ui.NewRDialog(win.QWidget)
		body := qt.NewQLabel3("Dialog body")
		body.SetStyleSheet("color: " + theme.CssColor(theme.ColorText) + ";")
		dialog.AddBodyWidget(body.QWidget)
		cancel := ui.NewRButton("Cancel", ui.BaseVariant, false)
		ok := ui.NewRButton("OK", ui.PrimaryVariant, false)
		cancel.OnClicked(dialog.Hide)
		ok.OnClicked(dialog.Hide)
		dialog.AddFooterWidget(cancel.QWidget)
		dialog.AddFooterWidget(ok.QWidget)
		dialog.OpenFor(win.QWidget)
	}

	dialogButton := ui.NewRButton("Open dialog", ui.SecondaryVariant, false)
	dialogButton.OnClicked(openDialog)
	panelLayout.AddWidget(dialogButton.QWidget)

	contentLayout.AddStretch()
	contentLayout.AddWidget(panel)
	contentLayout.SetAlignment(panel, qt.AlignHCenter)
	contentLayout.AddStretch()
	layout.AddContentWidget(content)

	layout.OnDestinationSelected(func(destination string) {
		routeStatus.SetText("Selected: " + destination)
	})
	layout.OnAddClicked(openDialog)
	layout.HeaderWidget.SearchField.OnTextChanged(func(text string) {
		if text == "" {
			routeStatus.SetText("Selected: " + layout.SidebarWidget.CurrentDestination())
			return
		}
		routeStatus.SetText("Search: " + text)
	})
}
