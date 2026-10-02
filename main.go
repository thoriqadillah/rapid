package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"rapid/db"
	"rapid/services/notification"
	"rapid/widget/app"
	"rapid/widget/components/downloads"
	"rapid/widget/theme"
	"rapid/widget/views"

	qt "github.com/mappu/miqt/qt6"
)

func main() {
	qt.NewQApplication(os.Args)
	qt.QGuiApplication_SetQuitOnLastWindowClosed(false)
	qt.QCoreApplication_SetApplicationName("rapid")
	dataPath := qt.QStandardPaths_WritableLocation(qt.QStandardPaths__AppDataLocation)
	os.MkdirAll(dataPath, 0o755)
	if err := db.Open(filepath.Join(dataPath, "database.db")); err != nil {
		panic(err)
	}
	defer db.Close()

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
	defer notifier.Close()

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
			Label: "New download",
			Callback: func() {
				downloadDialog := downloads.NewDownloadDialog(win.QWidget)
				downloadDialog.Open()
			},
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

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGABRT)
		<-sig
		qt.QCoreApplication_Quit()
	}()

	qt.QApplication_Exec()
}
