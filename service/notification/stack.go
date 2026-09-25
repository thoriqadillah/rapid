package notification

import (
	qt "github.com/mappu/miqt/qt6"
)

type Manager struct {
	Host *qt.QWidget

	active []*Popup
	closed bool
}

func NewManager(host *qt.QWidget) *Manager {
	return &Manager{Host: host}
}

func (m *Manager) Show(kind Kind, title, message string) *Popup {
	if m == nil || m.Host == nil || m.closed {
		return nil
	}
	popup := NewPopup(m.Host)
	popup.SetKind(kind)
	popup.SetTitle(title)
	popup.SetMessage(message)
	m.active = append(m.active, popup)
	popup.OnDismissed(func() {
		m.remove(popup)
	})
	popup.Open()
	m.Refresh()
	return popup
}

func (m *Manager) Refresh() {
	if m == nil || m.Host == nil {
		return
	}
	width := m.Host.Width()
	nextBottom := m.Host.Height() - popupBottom
	for _, popup := range m.active {
		if popup == nil || popup.IsHidden() {
			continue
		}
		popup.AdjustSize()
		popupW := popup.Width()
		if popupW <= 0 {
			popupW = popupWidth
		}
		y := nextBottom - popup.Height()
		popup.MoveTo(MoveTarget{X: width - popupRightMargin - popupW, Y: y, Width: popupW, Height: popup.Height()})
		popup.Raise()
		nextBottom = y - popupGap
	}
}

func (m *Manager) ActiveCount() int {
	if m == nil {
		return 0
	}
	return len(m.active)
}

func (m *Manager) Active() []*Popup {
	if m == nil {
		return nil
	}
	return append([]*Popup(nil), m.active...)
}

func (m *Manager) DismissAll() {
	if m == nil {
		return
	}
	for _, popup := range append([]*Popup(nil), m.active...) {
		if popup != nil {
			popup.Close()
		}
	}
	m.active = nil
}

func (m *Manager) Close() {
	if m == nil || m.closed {
		return
	}
	m.closed = true
	m.DismissAll()
}

func (m *Manager) remove(target *Popup) {
	if m == nil || target == nil {
		return
	}
	for index, popup := range m.active {
		if popup == target {
			m.active = append(m.active[:index], m.active[index+1:]...)
			target.Hide()
			target.DeleteLater()
			m.Refresh()
			return
		}
	}
}
