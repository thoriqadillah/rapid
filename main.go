package main

import (
	"os"

	"rapid/service/notification"
	"rapid/widget/app"
	"rapid/widget/components/downloads"
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
	win.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		super(event)
		event.Ignore()
		win.Hide()
	})

	notifier := notification.NewService(notification.Options{
		Host:       win.QWidget,
		EnableTray: true,
	})
	notifier.OnTrayActivated(func() {
		if win.IsVisible() && win.IsActiveWindow() {
			win.Hide()
			return
		}
		win.ShowNormal()
		win.Raise()
		win.ActivateWindow()
	})

	downloadDialog := downloads.NewDownloadDialog(win.QWidget)
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
			Callback: downloadDialog.Open,
		},
		{
			Label: "Quit",
			Callback: func() {
				qt.QCoreApplication_Quit()
			},
		},
	})

	stackView := qt.NewQStackedWidget2()
	win.SetCentralWidget(stackView.QWidget)

	navigation := app.NewNavigation(stackView)
	navigation.RegisterAll(app.RouteMap{
		app.RouteDownload: func(parent *qt.QWidget) *qt.QWidget {
			return views.NewDownloadView(parent, notifier, navigation)
		},
		app.RouteSettings: func(parent *qt.QWidget) *qt.QWidget {
			return views.NewSettingsView(parent, navigation)
		},
	})
	navigation.Replace(app.RouteDownload)

	win.Show()
	qt.QApplication_Exec()
}
