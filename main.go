package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"rapid/db"
	"rapid/services/download"
	"rapid/services/notification"
	"rapid/services/settings"
	"rapid/widget/app"
	"rapid/widget/components/downloads"
	"rapid/widget/theme"
	"rapid/widget/ui"
	"rapid/widget/views"

	qt "github.com/mappu/miqt/qt6"
	"github.com/mappu/miqt/qt6/mainthread"
)

func main() {
	if err := run(); err != nil {
		slog.Error("rapid exited with an error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetApplicationName("rapid")

	icon := qt.NewQIcon4(ui.IconPath("rapid.svg"))
	qt.QGuiApplication_SetWindowIcon(icon)
	icon.GoGC()

	trayAvailable := qt.QSystemTrayIcon_IsSystemTrayAvailable()
	qt.QGuiApplication_SetQuitOnLastWindowClosed(!trayAvailable)

	s := settings.Default(executableDir())
	if err := db.Open(filepath.Join(s.DataDir, "database.db")); err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	theme.Init()

	win := qt.NewQMainWindow2()
	win.SetWindowTitle("Rapid")
	win.Resize(1024, 700)
	win.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		if !trayAvailable {
			super(event) // QuitOnLastWindowClosed is true: closing quits
			return
		}
		event.Ignore()
		win.Hide()
	})

	showWindow := func() {
		win.ShowNormal()
		win.Raise()
		win.ActivateWindow()
	}

	notifier := notification.NewService(notification.Options{
		Host:       win.QWidget,
		EnableTray: trayAvailable,
	})
	defer notifier.Close()

	notifier.OnTrayActivated(func() {
		if win.IsVisible() && win.IsActiveWindow() {
			win.Hide()
			return
		}
		showWindow()
	})

	notifier.SetTrayMenu([]notification.MenuItem{
		{
			Label:    "Open",
			Callback: showWindow,
		},
		{
			Label: "New download",
			Callback: func() {
				showWindow()
				downloadDialog := downloads.NewDownloadDialog(win.QWidget)
				downloadDialog.Open()
			},
		},
		{
			Label:    "Quit",
			Callback: qt.QCoreApplication_Quit,
		},
	})

	stackView := qt.NewQStackedWidget2()
	win.SetCentralWidget(stackView.QWidget)

	downloadService := download.NewService()
	if err := downloadService.Refresh(context.Background()); err != nil {
		return fmt.Errorf("load downloads: %w", err)
	}

	navigation := app.NewNavigation(stackView)
	navigation.RegisterAll(app.RouteMap{
		app.RouteDownload: func(parent *qt.QWidget) *qt.QWidget {
			return views.NewDownloadView(parent, views.DownloadViewDeps{
				Navigation:     navigation,
				Notifier:       notifier,
				Service:        downloadService,
				PollIntervalMs: s.PollIntervalMs,
			})
		},
		app.RouteSettings: func(parent *qt.QWidget) *qt.QWidget {
			return views.NewSettingsView(parent, navigation)
		},
	})
	navigation.Replace(app.RouteDownload)

	win.Show()

	// SIGINT/SIGTERM quit cleanly on the GUI thread. SIGHUP is deliberately not
	// trapped: a terminal hangup should not be forced into the quit path.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		mainthread.Start(qt.QCoreApplication_Quit)
	}()

	qt.QApplication_Exec()
	return nil
}

func executableDir() string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return "."
	}
	return filepath.Dir(exe)
}
