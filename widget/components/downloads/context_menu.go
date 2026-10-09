package downloads

import (
	"rapid/services/download/api"

	qt "github.com/mappu/miqt/qt6"
)

// ItemContextMenu is the right-click menu for a download row. Enablement
// mirrors DownloadItemContextMenu.qml exactly.
type ItemContextMenu struct {
	*qt.QMenu

	item api.Download

	pauseAction  *qt.QAction
	resumeAction *qt.QAction
	stopAction   *qt.QAction
	deleteAction *qt.QAction
	copyAction   *qt.QAction
	openAction   *qt.QAction

	pauseCB  []func(string)
	resumeCB []func(string)
	stopCB   []func(string)
	deleteCB []func(api.Download)
	closedCB []func()
}

func NewItemContextMenu(parent *qt.QWidget) *ItemContextMenu {
	m := &ItemContextMenu{QMenu: qt.NewQMenu(parent)}
	m.SetFixedWidth(200)

	m.pauseAction = m.AddActionWithText("Pause")
	m.pauseAction.OnTriggered(func() { m.emitString(m.pauseCB) })

	m.resumeAction = m.AddActionWithText("Resume")
	m.resumeAction.OnTriggered(func() { m.emitString(m.resumeCB) })

	m.stopAction = m.AddActionWithText("Stop")
	m.stopAction.OnTriggered(func() { m.emitString(m.stopCB) })

	m.deleteAction = m.AddActionWithText("Delete")
	m.deleteAction.OnTriggered(func() {
		for _, fn := range m.deleteCB {
			if fn != nil {
				fn(m.item)
			}
		}
	})

	m.AddSeparator()

	m.copyAction = m.AddActionWithText("Copy URL")
	m.copyAction.OnTriggered(func() {
		if m.item.Resolved != nil && m.item.Resolved.URL != "" {
			qt.QGuiApplication_Clipboard().SetText(m.item.Resolved.URL)
		}
	})

	m.openAction = m.AddActionWithText("Go to file")
	m.openAction.OnTriggered(func() {
		if dir := m.item.FileDir(); dir != "" {
			qt.QDesktopServices_OpenUrl(qt.QUrl_FromLocalFile(dir))
		}
	})

	m.OnAboutToHide(func() {
		for _, fn := range m.closedCB {
			if fn != nil {
				fn()
			}
		}
	})

	return m
}

// ShowFor configures enablement for item and pops the menu at globalPos.
func (m *ItemContextMenu) ShowFor(item api.Download, globalPos *qt.QPoint) {
	if m == nil {
		return
	}
	m.applyState(item)
	m.Popup(globalPos)
}

// applyState sets the enablement matrix for item without showing the menu.
func (m *ItemContextMenu) applyState(item api.Download) {
	m.item = item

	m.pauseAction.SetEnabled(item.CanPause())
	m.resumeAction.SetEnabled(item.CanResume())
	m.stopAction.SetEnabled(item.CanPause())
	m.deleteAction.SetEnabled(item.IsFinished() || item.IsError())
	m.copyAction.SetEnabled(item.Resolved != nil && item.Resolved.URL != "")
	m.openAction.SetEnabled(item.FileDir() != "")
}

func (m *ItemContextMenu) OnPause(fn func(string)) {
	if fn != nil {
		m.pauseCB = append(m.pauseCB, fn)
	}
}

func (m *ItemContextMenu) OnResume(fn func(string)) {
	if fn != nil {
		m.resumeCB = append(m.resumeCB, fn)
	}
}

func (m *ItemContextMenu) OnStop(fn func(string)) {
	if fn != nil {
		m.stopCB = append(m.stopCB, fn)
	}
}

func (m *ItemContextMenu) OnDelete(fn func(api.Download)) {
	if fn != nil {
		m.deleteCB = append(m.deleteCB, fn)
	}
}

// OnClosed fires after the menu hides (the view resets its highlight state).
func (m *ItemContextMenu) OnClosed(fn func()) {
	if fn != nil {
		m.closedCB = append(m.closedCB, fn)
	}
}

func (m *ItemContextMenu) emitString(callbacks []func(string)) {
	for _, fn := range callbacks {
		if fn != nil {
			fn(m.item.GID)
		}
	}
}
