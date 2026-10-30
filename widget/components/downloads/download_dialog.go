package downloads

import (
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

func NewDownloadDialog(parent *qt.QWidget) *ui.RDialog {
	dialog := ui.NewRDialog(parent)
	dialog.SetWindowTitle("New download")
	dialog.SetMinimumWidth(500)

	dialog.AddBodyWidget(NewKVEditor(nil, "Headers").QWidget)
	dialog.AddBodyWidget(NewKVEditor(nil, "Cookies").QWidget)

	cancel := ui.NewRButton("Cancel", ui.BaseVariant, false)
	download := ui.NewRButton("Download", ui.PrimaryVariant, false)
	download.SetDisabled(true)

	close := func() {
		dialog.Hide()
		dialog.DeleteLater()
	}
	cancel.OnClicked(close)
	download.OnClicked(close)
	dialog.AddFooterWidget(cancel.QWidget)
	dialog.AddFooterWidget(download.QWidget)

	return dialog
}
