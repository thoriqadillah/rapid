package views

import (
	"context"
	"fmt"

	download "rapid/services/download"
	"rapid/services/download/api"
	"rapid/services/notification"
	"rapid/widget/app"
	"rapid/widget/components/downloads"
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

// DownloadViewDeps is the constructor injection that replaces the QML context
// properties (DownloadService/DownloadFilter/Clipboard/Navigation).
type DownloadViewDeps struct {
	Navigation     *app.Navigation
	Notifier       *notification.Service
	Service        *download.Service // nil -> empty DB-backed service
	PollIntervalMs int
}

func NewDownloadView(parent *qt.QWidget, deps DownloadViewDeps) *qt.QWidget {
	service := deps.Service
	ctx := context.Background()

	layout := downloads.NewLayout(deps.Navigation)

	banner := downloads.NewClipboardBanner(nil)

	panel := downloads.NewTablePanel(nil)

	header := downloads.NewTableHeader(panel)
	list := downloads.NewDownloadList(panel)

	panelLayout := qt.NewQVBoxLayout2()
	panelLayout.SetContentsMargins(0, 0, 0, 0)
	panelLayout.SetSpacing(0)
	panelLayout.AddWidget(header.QWidget)
	panelLayout.AddWidget2(list.QWidget, 1)
	panel.SetLayout(panelLayout.QLayout)

	content := qt.NewQWidget2()
	contentLayout := qt.NewQVBoxLayout2()
	contentLayout.SetContentsMargins(theme.SpacingSm, theme.SpacingSm, theme.SpacingSm, theme.SpacingSm)
	contentLayout.SetSpacing(theme.SpacingMd)
	contentLayout.AddWidget(banner.QWidget)
	contentLayout.AddWidget2(panel, 1)
	content.SetLayout(contentLayout.QLayout)
	layout.AddContentWidget(content)

	loadHistory := func(gid string) {
		if gid == "" {
			return
		}
		list.SetSpeedHistory(gid, service.SpeedSamples(ctx, gid))
	}

	render := func() {
		list.SetItems(service.ComputedItems())
		layout.SetCounts(service.Counts())
		loadHistory(list.ExpandedGid())
	}

	// The download service is a process singleton while this view is rebuilt on
	// every navigation. Keep the returned unsubscribe and detach on destroy so
	// the service does not retain this page's closures and widget tree forever.
	unsubscribe := service.OnChanged(render)
	layout.OnDestroyed(unsubscribe)

	list.OnExpand(loadHistory)

	// Phase 1/2: action buttons are intentionally not wired to the downloader.
	list.OnPause(func(gid string) { fmt.Printf("download: pause %s (not wired)\n", gid) })
	list.OnResume(func(gid string) { fmt.Printf("download: resume %s (not wired)\n", gid) })
	list.OnStop(func(gid string) { fmt.Printf("download: stop %s (not wired)\n", gid) })
	list.OnRemove(func(gid string, deleteFromDisk bool) {
		fmt.Printf("download: remove %s fromDisk=%v (not wired)\n", gid, deleteFromDisk)
	})
	list.OnDeleteRequested(func(d api.Download) {
		dialog := downloads.NewDeleteConfirmationDialog(parent, d.Name(), func(deleteFromDisk bool) {
			fmt.Printf("download: delete %s fromDisk=%v (not wired)\n", d.GID, deleteFromDisk)
		})
		dialog.Open()
	})

	layout.OnDestinationSelected(func(destination string) {
		if destination == "" {
			return
		}
		list.Collapse()
		service.SetCategory(destination)
	})
	layout.HeaderWidget.SearchField.OnTextChanged(func(text string) {
		list.Collapse()
		service.SetSearch(text)
	})
	layout.OnAddClicked(func() {
		openDownloadDialog(parent, deps)
	})

	banner.OnDownload(func(string) { openDownloadDialog(parent, deps) })
	banner.OnDismiss(func() { banner.SetURL("") })

	render()

	// Load the DB rows (notifies -> render); a closed/missing DB just keeps
	// the empty state instead of falling back to in-memory dummies.
	service.Refresh(ctx)

	if deps.PollIntervalMs > 0 {
		timer := qt.NewQTimer2(layout.QObject)
		timer.SetInterval(deps.PollIntervalMs)
		timer.OnTimeout(func() { service.Refresh(ctx) })
		timer.Start2()
	}

	return layout.QWidget
}

func openDownloadDialog(parent *qt.QWidget, deps DownloadViewDeps) {
	dialog := downloads.NewDownloadDialog(parent, downloads.DownloadDialogOptions{Navigation: deps.Navigation})
	dialog.Open()
}
