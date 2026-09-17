package ui

import (
	"fmt"

	"rapid/theme"

	qt "github.com/mappu/miqt/qt6"
)

const maxWidgetHeight = 16777215

type RDialog struct {
	*qt.QDialog
	BodyLayout   *qt.QVBoxLayout
	FooterLayout *qt.QHBoxLayout

	owner     *qt.QWidget
	overlay   *qt.QWidget
	maxHeight int
	opened    []func()
}

func NewRDialog(owner *qt.QWidget) *RDialog {
	d := &RDialog{QDialog: qt.NewQDialog(owner), owner: owner}
	d.SetMinimumWidth(500)
	d.SetModal(true)
	d.SetWindowModality(qt.WindowModal)
	d.SetStyleSheet(fmt.Sprintf(`
		QDialog {
			background-color: %s;
			color: %s;
			font-size: %dpx;
		}
	`, theme.CssColor(theme.ColorBackground), theme.CssColor(theme.ColorText), theme.TextSize))

	box := qt.NewQVBoxLayout2()
	box.SetContentsMargins(theme.SpacingSm, theme.SpacingSm, theme.SpacingSm, theme.SpacingSm)
	d.QDialog.SetLayout(box.QLayout)

	esc := qt.NewQShortcut2(qt.NewQKeySequence2("Esc"), d.QDialog.QObject)
	esc.OnActivated(d.Reject)

	d.BodyLayout = qt.NewQVBoxLayout2()
	d.BodyLayout.SetContentsMargins(0, 0, 0, 0)
	d.BodyLayout.SetSpacing(theme.SpacingMd)
	box.AddLayout(d.BodyLayout.QLayout)

	d.FooterLayout = qt.NewQHBoxLayout2()
	d.FooterLayout.SetContentsMargins(0, 0, 0, 0)
	d.FooterLayout.SetSpacing(theme.SpacingMd)
	d.FooterLayout.AddStretch()
	box.AddLayout(d.FooterLayout.QLayout)

	d.OnHideEvent(func(super func(*qt.QHideEvent), event *qt.QHideEvent) {
		super(event)
		d.cleanupOverlay()
	})
	d.OnDestroyed(func() {
		d.cleanupOverlay()
	})
	return d
}

// OpenFor shows the dialog with an owner-sized modal overlay and emits the
// lightweight opened callbacks after the dialog has been raised and focused.
//
// MIQT only permits virtual event overrides on Go objects that directly own
// the overridden Qt instance. The owner is supplied by the caller, so overlay
// tracking deliberately happens at open time instead of installing resize or
// close overrides on that arbitrary widget.
func (d *RDialog) OpenFor(owner *qt.QWidget) {
	if owner != nil && owner != d.owner {
		d.owner = owner
		d.SetParent2(owner, qt.Dialog)
	}
	if d.owner != nil {
		d.ensureOverlay()
		d.RefreshOverlay()
		d.overlay.Show()
		d.overlay.Raise()
	}
	d.Show()
	d.Raise()
	d.ActivateWindow()
	d.SetFocus()
	for _, callback := range d.opened {
		if callback != nil {
			callback()
		}
	}
}

func (d *RDialog) OnOpened(fn func()) {
	if fn != nil {
		d.opened = append(d.opened, fn)
	}
}

func (d *RDialog) SetMaxHeight(height int) {
	if height < 0 {
		height = 0
	}
	d.maxHeight = height
	if height == 0 {
		d.SetMaximumHeight(maxWidgetHeight)
	} else {
		d.SetMaximumHeight(height)
	}
}

func (d *RDialog) MaxHeight() int {
	return d.maxHeight
}

func (d *RDialog) AddBodyWidget(widget *qt.QWidget) {
	if widget != nil {
		d.BodyLayout.AddWidget(widget)
	}
}

func (d *RDialog) AddFooterWidget(widget *qt.QWidget) {
	if widget != nil {
		d.FooterLayout.AddWidget(widget)
	}
}

func (d *RDialog) OverlayWidget() *qt.QWidget {
	return d.overlay
}

func (d *RDialog) ensureOverlay() {
	if d.owner == nil {
		return
	}
	if d.overlay == nil {
		d.overlay = qt.NewQWidget3(d.owner, 0)
		d.overlay.SetStyleSheet("background-color: rgba(0, 0, 0, 115);")
		d.overlay.SetToolTip("")
	}
}

// RefreshOverlay resizes the modal overlay to the current owner geometry.
// Call it after the owner is resized while the dialog is visible.
func (d *RDialog) RefreshOverlay() {
	if d.overlay != nil && d.owner != nil {
		d.overlay.SetGeometryWithGeometry(d.owner.Rect())
	}
}

func (d *RDialog) cleanupOverlay() {
	if d.overlay != nil {
		d.overlay.Hide()
	}
}
