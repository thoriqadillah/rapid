package downloads

import (
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

func NewDownloadDialog(parent *qt.QWidget) *ui.RDialog {
	dialog := ui.NewRDialog(parent)
	body := qt.NewQLabel3("Dialog body")
	body.SetStyleSheet("color: " + theme.CssColor(theme.ColorText) + ";")
	dialog.AddBodyWidget(body.QWidget)
	cancel := ui.NewRButton("Cancel", ui.BaseVariant, false)
	ok := ui.NewRButton("OK", ui.PrimaryVariant, false)

	close := func() {
		dialog.Hide()
		dialog.DeleteLater()
	}

	cancel.OnClicked(close)
	ok.OnClicked(close)
	dialog.AddFooterWidget(cancel.QWidget)
	dialog.AddFooterWidget(ok.QWidget)

	return dialog
}
