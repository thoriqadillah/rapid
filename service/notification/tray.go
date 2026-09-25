package notification

import (
	"fmt"

	qt "github.com/mappu/miqt/qt6"
)

// TrayOptions makes the service testable without requiring a desktop tray.
type TrayOptions struct {
	Icon     *qt.QIcon
	TrayIcon *qt.QSystemTrayIcon
	Menu     *qt.QMenu
	Items    []MenuItem
	Enable   bool
}

type trayController struct {
	icon     *qt.QIcon
	tray     *qt.QSystemTrayIcon
	menu     *qt.QMenu
	items    []MenuItem
	actions  []*qt.QAction
	ownsMenu bool
	closed   bool

	activatedCallbacks []func()
}

func newTray(options TrayOptions) *trayController {
	if !options.Enable && options.TrayIcon == nil {
		return nil
	}
	icon := options.Icon
	if icon == nil {
		icon = qt.NewQIcon4("")
	}
	tray := options.TrayIcon
	if tray == nil {
		tray = qt.NewQSystemTrayIcon2(icon)
	}
	menu := options.Menu
	ownsMenu := menu == nil
	if menu == nil {
		menu = qt.NewQMenu2()
	}
	controller := &trayController{
		icon:     icon,
		tray:     tray,
		menu:     menu,
		items:    options.Items,
		ownsMenu: ownsMenu,
	}
	controller.rebuild(options.Items)
	tray.SetToolTip("Rapid")
	tray.SetContextMenu(menu)
	tray.OnActivated(func(reason qt.QSystemTrayIcon__ActivationReason) {
		if reason == qt.QSystemTrayIcon__Trigger || reason == qt.QSystemTrayIcon__DoubleClick {
			for _, fn := range controller.activatedCallbacks {
				if fn != nil {
					fn()
				}
			}
		}
	})
	tray.Show()
	return controller
}

func (t *trayController) onActivated(fn func()) {
	if fn != nil {
		t.activatedCallbacks = append(t.activatedCallbacks, fn)
	}
}

func (t *trayController) rebuild(items []MenuItem) {
	if items == nil {
		return
	}
	t.items = items
	t.menu.Clear()
	t.actions = t.actions[:0]
	for _, item := range items {
		action := t.menu.AddActionWithText(item.Label)
		action.OnTriggered(item.Callback)
		t.actions = append(t.actions, action)
	}
}

func (t *trayController) showMessage(title, message string, icon *qt.QIcon) {
	if t == nil || t.tray == nil || t.closed {
		return
	}
	if icon == nil {
		icon = t.icon
	}
	t.tray.ShowMessage3(title, message, icon, 5000)
}

func (t *trayController) setBadge(count int) {
	if t == nil || t.tray == nil || t.closed {
		return
	}
	if count <= 0 {
		t.tray.SetIcon(t.icon)
		return
	}
	icon := t.icon
	if icon == nil || icon.IsNull() {
		return
	}
	size := icon.ActualSize(qt.NewQSize2(64, 64))
	if size == nil || size.Width() <= 0 || size.Height() <= 0 {
		size = qt.NewQSize2(64, 64)
	}
	pixmap := qt.NewQPixmap3(size)
	pixmap.FillWithFillColor(qt.NewQColor11(0, 0, 0, 0))
	painter := qt.NewQPainter2(pixmap.QPaintDevice)
	painter.SetRenderHint2(qt.QPainter__Antialiasing, true)
	icon.Paint(painter, qt.NewQRect4(0, 0, size.Width(), size.Height()))
	badge := fmtBadge(count)
	badgeSize := size.Width() / 2
	if badgeSize < 10 {
		badgeSize = 10
	}
	width := badgeSize
	if len(badge) > 1 {
		width += 6
	}
	if width < badgeSize {
		width = badgeSize
	}
	height := badgeSize
	painter.SetPen(qt.NewQColor6("white"))
	painter.SetBrush(qt.NewQBrush3(qt.NewQColor6("#e53935")))
	painter.DrawRoundedRect2(size.Width()-width, size.Height()-height, width, height, 4, 4)
	font := qt.NewQFont()
	font.SetPixelSize(badgeSize * 3 / 4)
	font.SetBold(true)
	painter.SetFont(font)
	painter.DrawText7(size.Width()-width, size.Height()-height, width, height, 0x84, badge)
	painter.End()
	t.tray.SetIcon(qt.NewQIcon2(pixmap))
}

func fmtBadge(count int) string {
	if count >= 100 {
		return "99+"
	}
	return fmt.Sprintf("%d", count)
}

func (t *trayController) close() {
	if t == nil || t.closed {
		return
	}
	t.closed = true
	if t.tray != nil {
		t.tray.Hide()
	}
	if t.menu != nil {
		t.menu.Close()
		if t.ownsMenu {
			t.menu.DeleteLater()
		}
	}
}
