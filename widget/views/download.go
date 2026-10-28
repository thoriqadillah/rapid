package views

import (
	"context"

	download "rapid/services/download"
	"rapid/services/download/api"
	"rapid/services/notification"
	"rapid/widget/app"
	"rapid/widget/components/downloads"
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

type DownloadViewDeps struct {
	Navigation *app.Navigation
	Notifier   *notification.Service
	Service    *download.Service
}

func NewDownloadView(parent *qt.QWidget, deps DownloadViewDeps) *qt.QWidget {
	service := deps.Service
	ctx := context.Background()

	layout := downloads.NewLayout(deps.Navigation)
	layout.BindCounts(service.Counts)
	layout.BindSearch(service.Search)
	layout.BindDestination(service.Category)
	layout.OnAddClicked(func() {
		openDownloadDialog(parent, deps)
	})

	banner := downloads.NewClipboardBanner(nil)

	panel := downloads.NewTablePanel(nil)

	header := downloads.NewTableHeader(panel)

	list := downloads.NewDownloadList(panel)
	list.BindItems(service.Items)
	list.BindFilterChange(service.FilterChanged)
	list.OnExpand(func(gid string) {
		list.SetSpeedHistory(gid, service.SpeedSamples(ctx, gid))
	})
	list.OnPause(func(gid string) { service.Pause(ctx, gid) })
	list.OnResume(func(gid string) { service.Resume(ctx, gid) })
	list.OnStop(func(gid string) { service.Stop(ctx, gid) })
	list.OnRemove(func(gid string, deleteFromDisk bool) {
		service.Delete(ctx, gid, deleteFromDisk)
	})
	list.OnDeleteRequested(func(d api.Download) {
		dialog := downloads.NewDeleteConfirmationDialog(parent, d.Name(), func(deleteFromDisk bool) {
			service.Delete(ctx, d.GID, deleteFromDisk)
		})
		dialog.Open()
	})

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

	banner.OnDownload(func(string) { openDownloadDialog(parent, deps) })
	banner.OnDismiss(func() { banner.SetURL("") })

	return layout.QWidget
}

func openDownloadDialog(parent *qt.QWidget, deps DownloadViewDeps) {
	dialog := downloads.NewDownloadDialog(parent, downloads.DownloadDialogOptions{Navigation: deps.Navigation})
	dialog.Open()
}
