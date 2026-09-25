package main

import (
	"fmt"
	"os"

	"rapid/service/notification"
	"rapid/widget/theme"
	"rapid/widget/views"

	qt "github.com/mappu/miqt/qt6"
)

func main() {
	qt.NewQApplication(os.Args)
	qt.QGuiApplication_SetQuitOnLastWindowClosed(false)
	qt.QCoreApplication_SetApplicationName("Rapid")
	theme.Init()

	win := qt.NewQMainWindow2()
	win.SetWindowTitle("Rapid")
	win.Resize(1024, 700)
	// Close button hides to tray instead of quitting; use the tray "Quit" item.
	win.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		super(event)
		event.Ignore()
		win.Hide()
	})
	win.SetStyleSheet(fmt.Sprintf(`
		QMainWindow { color: %s; font-size: %dpx; }
	`, theme.CssColor(theme.ColorText), theme.TextSize))

	notifier := notification.NewService(notification.Options{
		Host:       win.QWidget,
		EnableTray: true,
	})

	downloadView := views.NewDownloadView(win, notifier)
	win.SetCentralWidget(downloadView.Widget)

	notifier.OnTrayActivated(func() {
		if win.IsVisible() && win.IsActiveWindow() {
			win.Hide()
			return
		}
		win.ShowNormal()
		win.Raise()
		win.ActivateWindow()
	})
	notifier.SetTrayMenu([]notification.MenuItem{
		{
			Label: "Open",
			Callback: func() {
				win.ShowNormal()
				win.Raise()
				win.ActivateWindow()
			},
		},
		{
			Label:    "New download",
			Callback: downloadView.Add,
		},
		{
			Label: "Quit",
			Callback: func() {
				qt.QCoreApplication_Quit()
			},
		},
	})

	win.Show()
	qt.QApplication_Exec()
}
