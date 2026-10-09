package downloads

import (
	"rapid/widget/app"
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

// DownloadDialogOptions is the deps-ready shape for the deferred full
// DownloadDialog port. The resolve/download flow, ResolverUri, KVEditor and
// AdvancedOptions are explicitly out of scope here; this stub only fixes the
// constructor so call sites don't churn later.
type DownloadDialogOptions struct {
	Navigation *app.Navigation
}

// NewDownloadDialog builds the "New download" stub. Full port is deferred.
func NewDownloadDialog(parent *qt.QWidget, options ...DownloadDialogOptions) *ui.RDialog {
	dialog := ui.NewRDialog(parent)
	dialog.SetWindowTitle("New download")
	dialog.SetMinimumWidth(500)

	body := qt.NewQLabel3("Resolve and download UI is not ported yet.")
	body.SetWordWrap(true)
	body.SetStyleSheet("color: " + theme.CssColor(theme.ColorTextMuted) + ";")
	dialog.AddBodyWidget(body.QWidget)

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
