package notification

import (
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

type Options struct {
	Host             *qt.QWidget
	TrayIcon         *qt.QSystemTrayIcon
	TrayMenu         *qt.QMenu
	NotificationIcon *qt.QIcon
	EnableTray       bool
}

type Service struct {
	Manager *Manager
	tray    *trayController
	closed  bool

	notificationCallbacks []func(Event)
}

type MenuItem struct {
	Label    string
	Callback func()
}

type MenuItems map[string]MenuItem

func NewService(options Options) *Service {
	s := &Service{}
	if options.Host != nil {
		s.Manager = NewManager(options.Host)
	}
	if options.NotificationIcon == nil {
		options.NotificationIcon = qt.NewQIcon4(ui.IconPath("rapid.svg"))
	}
	if options.EnableTray || options.TrayIcon != nil {
		s.tray = newTray(TrayOptions{
			Icon:     options.NotificationIcon,
			TrayIcon: options.TrayIcon,
			Menu:     options.TrayMenu,
			Enable:   options.EnableTray,
		})
	}

	return s
}

// SetTrayMenu rebuilds the tray context menu from the given map.
func (s *Service) SetTrayMenu(items []MenuItem) {
	if s == nil || s.tray == nil {
		return
	}
	s.tray.rebuild(items)
}

// OnTrayActivated registers a handler for clicks on the tray icon itself
// (Trigger/DoubleClick), independent of the context menu items.
func (s *Service) OnTrayActivated(fn func()) {
	if s == nil || s.tray == nil {
		return
	}
	s.tray.onActivated(fn)
}

func (s *Service) OnOpen(fn func(Event)) {
	if s != nil && fn != nil {
		s.notificationCallbacks = append(s.notificationCallbacks, fn)
	}
}

func (s *Service) Error(title, message string, shouldNotify bool) {
	s.publish(KindError, title, message, shouldNotify)
}

func (s *Service) Success(title, message string, shouldNotify bool) {
	s.publish(KindSuccess, title, message, shouldNotify)
}

func (s *Service) Info(title, message string, shouldNotify bool) {
	s.publish(KindInfo, title, message, shouldNotify)
}

func (s *Service) publish(kind Kind, title, message string, shouldNotify bool) {
	if s == nil || s.closed {
		return
	}
	event := Event{Kind: kind, Title: title, Message: message}
	for _, fn := range s.notificationCallbacks {
		if fn != nil {
			fn(event)
		}
	}

	if s.Manager != nil {
		s.Manager.Show(kind, title, message)
	}

	if shouldNotify && s.tray != nil {
		s.tray.showMessage(title, message, s.tray.icon)
	}
}

func (s *Service) ActiveCount() int {
	if s == nil || s.Manager == nil {
		return 0
	}
	return s.Manager.ActiveCount()
}

func (s *Service) Close() {
	if s == nil || s.closed {
		return
	}
	s.closed = true
	if s.Manager != nil {
		s.Manager.Close()
	}
	if s.tray != nil {
		s.tray.close()
	}
}

func KindColor(kind Kind) *qt.QColor {
	switch kind {
	case KindError:
		return theme.ColorDanger
	case KindSuccess:
		return theme.ColorSuccess
	default:
		return theme.ColorInfo
	}
}
