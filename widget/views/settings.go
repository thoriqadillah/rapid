package views

import (
	"rapid/widget/app"
	"rapid/widget/components/settings"
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

func NewSettingsView(parent *qt.QWidget, navigation *app.Navigation) *qt.QWidget {
	layout := settings.NewLayout(navigation)

	content := qt.NewQWidget2()
	contentLayout := qt.NewQVBoxLayout2()
	contentLayout.SetContentsMargins(theme.SpacingXl, theme.SpacingXl, theme.SpacingXl, theme.SpacingXl)
	contentLayout.SetSpacing(theme.SpacingLg)
	content.SetLayout(contentLayout.QLayout)

	title := qt.NewQLabel3("Settings")
	title.SetStyleSheet("font-size: 20px; color: " + theme.CssColor(theme.ColorText) + ";")
	contentLayout.AddWidget(title.QWidget)
	copy := qt.NewQLabel3("Settings page placeholder")
	copy.SetStyleSheet("color: " + theme.CssColor(theme.ColorTextMuted) + ";")
	contentLayout.AddWidget(copy.QWidget)
	contentLayout.AddStretch()

	layout.AddContentWidget(content)

	return layout.QWidget
}
