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
	// setBadge runs on every count change; release every Qt temporary so the
	// tray badge does not leak a painter, pixmap, icon, font and several colors
	// per update. MIQT does not finalize New* constructor results.
	requested := qt.NewQSize2(64, 64)
	size := icon.ActualSize(requested)
	requested.Delete()
	if size == nil || size.Width() <= 0 || size.Height() <= 0 {
		fallback := qt.NewQSize2(64, 64)
		size = fallback
		defer fallback.Delete()
	}

	pixmap := qt.NewQPixmap3(size)
	defer pixmap.Delete()
	clear := qt.NewQColor11(0, 0, 0, 0)
	pixmap.FillWithFillColor(clear)
	clear.Delete()

	painter := qt.NewQPainter2(pixmap.QPaintDevice)
	defer painter.Delete()
	painter.SetRenderHint2(qt.QPainter__Antialiasing, true)
	rect := qt.NewQRect4(0, 0, size.Width(), size.Height())
	icon.Paint(painter, rect)
	rect.Delete()

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
	penColor := qt.NewQColor6("white")
	badgeColor := qt.NewQColor6("#e53935")
	brush := qt.NewQBrush3(badgeColor)
	painter.SetPen(penColor)
	painter.SetBrush(brush)
	painter.DrawRoundedRect2(size.Width()-width, size.Height()-height, width, height, 4, 4)
	penColor.Delete()
	badgeColor.Delete()
	brush.Delete()

	font := qt.NewQFont()
	font.SetPixelSize(badgeSize * 3 / 4)
	font.SetBold(true)
	painter.SetFont(font) // QPainter copies the font
	font.Delete()
	painter.DrawText7(size.Width()-width, size.Height()-height, width, height, 0x84, badge)

	badgeIcon := qt.NewQIcon2(pixmap)
	t.tray.SetIcon(badgeIcon) // tray copies the icon
	badgeIcon.Delete()
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
