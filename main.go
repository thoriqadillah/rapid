package main

import (
	"fmt"
	"os"

	"rapid/theme"
	"rapid/ui"

	qt "github.com/mappu/miqt/qt6"
)

func main() {
	qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetApplicationName("Rapid")
	theme.Init()

	win := qt.NewQMainWindow2()
	win.SetWindowTitle("Rapid")
	win.Resize(600, 480)
	win.SetStyleSheet(fmt.Sprintf(`
		QMainWindow {
			color: %s;
			font-size: %dpx;
		}
	`, theme.CssColor(theme.ColorText), theme.TextSize))

	central := qt.NewQWidget2()
	win.SetCentralWidget(central)
	root := qt.NewQVBoxLayout2()
	root.SetContentsMargins(theme.SpacingXl, theme.SpacingXl, theme.SpacingXl, theme.SpacingXl)
	root.SetSpacing(theme.SpacingLg)
	central.SetLayout(root.QLayout)

	title := qt.NewQLabel3("component demo")
	title.SetStyleSheet("font-size: 20px; color: " + theme.CssColor(theme.ColorText) + ";")
	root.AddWidget(title.QWidget)

	buttons := qt.NewQHBoxLayout2()
	buttons.SetSpacing(theme.SpacingSm)
	root.AddLayout(buttons.QLayout)
	buttons.AddWidget(ui.NewRButton("Base", ui.BaseVariant, false).QWidget)
	buttons.AddWidget(ui.NewRButton("Primary", ui.PrimaryVariant, false).QWidget)
	buttons.AddWidget(ui.NewRButton("Danger", ui.DangerVariant, false).QWidget)
	buttons.AddWidget(ui.NewRButton("Link", ui.LinkVariant, false).QWidget)
	buttons.AddWidget(ui.NewRButton("Outline", ui.PrimaryVariant, true).QWidget)
	buttons.AddWidget(ui.NewRButtonIcon("MdiLightPlus.svg", ui.BaseVariant, false).QWidget)

	iconBtn := ui.NewRButton("New", ui.PrimaryVariant, false)
	iconBtn.SetIconSource(ui.IconPath("MdiLightPlus.svg"))
	buttons.AddWidget(iconBtn.QWidget)

	disabledBtn := ui.NewRButton("Disabled", ui.PrimaryVariant, false)
	disabledBtn.SetDisabled(true)
	buttons.AddWidget(disabledBtn.QWidget)

	root.AddWidget(ui.NewRSwitch(true).QWidget)

	field := ui.NewRTextField()
	field.SetLabel("URL")
	field.SetPlaceholder("https://example.com")
	field.SetError("invalid URL")
	field.SetPrefixIcon(ui.IconPath("MdiLightContentPaste.svg"))
	root.AddWidget(field.QWidget)

	dialogBtn := ui.NewRButton("Open dialog", ui.SecondaryVariant, false)
	dialogBtn.OnClicked(func() {
		dialog := ui.NewRDialog(win.QWidget)
		body := qt.NewQLabel3("Dialog body")
		body.SetStyleSheet("color: " + theme.CssColor(theme.ColorText) + ";")
		dialog.BodyLayout.AddWidget(body.QWidget)
		buttons := qt.NewQHBoxLayout2()
		buttons.SetSpacing(theme.SpacingSm)
		cancelBtn := ui.NewRButton("Cancel", ui.BaseVariant, false)
		okBtn := ui.NewRButton("OK", ui.PrimaryVariant, false)

		buttons.AddWidget(cancelBtn.QWidget)
		buttons.AddWidget(okBtn.QWidget)
		dialog.FooterLayout.AddLayout(buttons.QLayout)
		dialog.Show()
	})
	root.AddWidget(dialogBtn.QWidget)

	win.Show()
	qt.QApplication_Exec()
}
