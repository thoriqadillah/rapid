package downloads

import (
	"fmt"

	"rapid/lib/helpers/bools"
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

// NewDeleteConfirmationDialog builds the "Delete download" confirm dialog with
// a default-checked From-disk switch. onConfirm receives the switch state.
func NewDeleteConfirmationDialog(owner *qt.QWidget, displayName string, onConfirm func(bool)) *ui.RDialog {
	dialog := ui.NewRDialog(owner)
	dialog.SetWindowTitle("Delete download")
	dialog.SetAttribute(qt.WA_DeleteOnClose)

	name := bools.Ternary(displayName != "", displayName, "this download")
	message := styledLabel(fmt.Sprintf("Remove %s forever?", name), theme.ColorText)
	message.SetWordWrap(true)
	dialog.AddBodyWidget(message.QWidget)

	fromDisk := ui.NewRSwitch(true)
	fromDisk.SetAccessibleName("Delete from disk")
	fromDiskLabel := styledLabel("From disk", theme.ColorText)
	fromDiskRow := qt.NewQWidget3(nil, 0)
	fromDiskLayout := qt.NewQHBoxLayout2()
	fromDiskLayout.SetContentsMargins(0, 0, 0, 0)
	fromDiskLayout.SetSpacing(theme.SpacingSm)
	fromDiskLayout.AddWidget(fromDisk.QWidget)
	fromDiskLayout.AddWidget(fromDiskLabel.QWidget)
	fromDiskLayout.AddStretch()
	fromDiskRow.SetLayout(fromDiskLayout.QLayout)
	dialog.FooterLayout.InsertWidget(0, fromDiskRow)

	confirm := func() {
		deleteFromDisk := fromDisk.IsChecked()
		if onConfirm != nil {
			onConfirm(deleteFromDisk)
		}
		dialog.Accept()
	}

	// QDialog already handles Esc (via RDialog) and Enter (default button), and
	// the old Return/Enter shortcuts fired the destructive confirm regardless of
	// focus — including while Cancel was focused.
	cancel := ui.NewRButton("Cancel", ui.BaseVariant, false)
	cancel.SetAutoDefault(false)
	cancel.OnClicked(dialog.Reject)
	del := ui.NewRButton("Delete", ui.DangerVariant, false)
	del.SetDefault(true)
	del.OnClicked(confirm)
	dialog.AddFooterWidget(cancel.QWidget)
	dialog.AddFooterWidget(del.QWidget)
	cancel.SetFocus()

	return dialog
}
