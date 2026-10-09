package downloads

import (
	"rapid/lib"
	"rapid/lib/helpers/bools"
	"rapid/services/download/api"
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

const detailLabelWidth = 100

// DownloadItemDetail is the panel shown under an expanded row: resolver card,
// speed sparkline, progress, file/mime/status metadata, error and actions.
// Actions fire callbacks; no service calls happen here.
type DownloadItemDetail struct {
	*qt.QWidget

	item api.Download

	resolver     *resolvedUriPanel
	chart        *SpeedChart
	progress     *ProgressBar
	progressText *qt.QLabel
	fileDir      *elidedLabel
	mimeType     *qt.QLabel
	status       *qt.QLabel
	errorRow     *qt.QWidget
	errorLabel   *qt.QLabel
	pauseBtn     *ui.RButton
	stopBtn      *ui.RButton
	removeBtn    *ui.RButton

	pauseCB  []func(string)
	resumeCB []func(string)
	stopCB   []func(string)
	removeCB []func(string, bool)
}

func NewDownloadItemDetail(parent *qt.QWidget) *DownloadItemDetail {
	d := &DownloadItemDetail{QWidget: qt.NewQWidget3(parent, 0)}

	column := qt.NewQVBoxLayout2()
	column.SetContentsMargins(theme.SpacingLg, theme.SpacingMd, theme.SpacingLg, theme.SpacingLg)
	column.SetSpacing(theme.SpacingMd)

	d.resolver = newResolverPanel(d.QWidget)
	column.AddWidget(d.resolver.QWidget)

	d.chart = NewSpeedChart(d.QWidget)
	column.AddWidget(d.chart.QWidget)

	d.progress = NewProgressBar(d.QWidget)
	d.progressText = styledLabel("0%", theme.ColorTextMuted)
	progressRow := qt.NewQHBoxLayout2()
	progressRow.SetContentsMargins(0, 0, 0, 0)
	progressRow.SetSpacing(theme.SpacingMd)
	progressRow.AddWidget(d.label("Progress").QWidget)
	progressRow.AddWidget2(d.progress.QWidget, 1)
	progressRow.AddWidget(d.progressText.QWidget)
	column.AddLayout(progressRow.QLayout)

	d.fileDir = newElidedLabel("—", theme.ColorText, qt.ElideMiddle)
	column.AddWidget(metaRow(d.label("File location").QWidget, d.fileDir.QWidget))

	d.mimeType = styledLabel("—", theme.ColorText)
	column.AddWidget(metaRow(d.label("MIME type").QWidget, d.mimeType.QWidget))

	d.status = styledLabel("", theme.ColorText)
	column.AddWidget(metaRow(d.label("Status").QWidget, d.status.QWidget))

	d.errorLabel = styledLabel("", theme.ColorDanger)
	d.errorLabel.SetWordWrap(true)
	d.errorRow = metaRow(d.label("Error").QWidget, d.errorLabel.QWidget)
	d.errorRow.SetVisible(false)
	column.AddWidget(d.errorRow)

	d.pauseBtn = ui.NewRButton("Pause", ui.SecondaryVariant, false)
	d.pauseBtn.SetIconSource(ui.IconPath("MdiLightPause.svg"))
	d.pauseBtn.OnClicked(func() {
		if d.item.CanResume() {
			d.emit(d.resumeCB, d.item.GID)
			return
		}
		d.emit(d.pauseCB, d.item.GID)
	})

	d.stopBtn = ui.NewRButton("Stop", ui.BaseVariant, false)
	d.stopBtn.SetIconSource(ui.IconPath("MdiLightStop.svg"))
	d.stopBtn.OnClicked(func() { d.emit(d.stopCB, d.item.GID) })

	d.removeBtn = ui.NewRButtonIcon("MaterialSymbolsLightDeleteForeverOutlineSharp.svg", ui.DangerVariant, false)
	d.removeBtn.SetTooltip("Remove the file forever")
	d.removeBtn.SetAccessibleName("Remove download")
	d.removeBtn.OnClicked(func() { d.openDeleteDialog() })

	actions := qt.NewQHBoxLayout2()
	actions.SetContentsMargins(0, 0, 0, 0)
	actions.SetSpacing(theme.SpacingSm)
	actions.AddWidget(d.pauseBtn.QWidget)
	actions.AddWidget(d.stopBtn.QWidget)
	actions.AddStretch()
	actions.AddWidget(d.removeBtn.QWidget)
	column.AddLayout(actions.QLayout)

	d.SetLayout(column.QLayout)
	return d
}

func (d *DownloadItemDetail) label(text string) *qt.QLabel {
	l := styledLabel(text, theme.ColorTextMuted)
	l.SetMinimumWidth(detailLabelWidth)
	return l
}

// metaRow builds "label  value(fill)".
func metaRow(label, value *qt.QWidget) *qt.QWidget {
	row := qt.NewQWidget3(nil, 0)
	layout := qt.NewQHBoxLayout2()
	layout.SetContentsMargins(0, 0, 0, 0)
	layout.SetSpacing(theme.SpacingMd)
	layout.AddWidget(label)
	layout.AddWidget2(value, 1)
	row.SetLayout(layout.QLayout)
	return row
}

func (d *DownloadItemDetail) emit(callbacks []func(string), gid string) {
	for _, fn := range callbacks {
		if fn != nil {
			fn(gid)
		}
	}
}

func (d *DownloadItemDetail) openDeleteDialog() {
	dialog := NewDeleteConfirmationDialog(d.QWidget, d.item.Name(), func(deleteFromDisk bool) {
		for _, fn := range d.removeCB {
			if fn != nil {
				fn(d.item.GID, deleteFromDisk)
			}
		}
	})
	dialog.Open()
}

func (d *DownloadItemDetail) SetItem(item api.Download) {
	d.item = item
	d.resolver.SetItem(item)
	categoryColor := theme.CategoryColor(item.Category)
	d.chart.SetCategory(item.Category)
	d.progress.SetValue(item.Progress(), categoryColor, false)
	d.progressText.SetText(item.FormatProgress())

	dir := item.FileDir()
	d.fileDir.SetFullText(bools.Ternary(dir != "", dir, "—"))

	mime := "—"
	if item.Resolved != nil && item.Resolved.MIMEType != "" {
		mime = item.Resolved.MIMEType
	}
	d.mimeType.SetText(mime)
	d.status.SetText(item.FormatStatus())

	hasError := item.ErrorMessage != "" && item.IsError()
	d.errorRow.SetVisible(hasError)
	if hasError {
		d.errorLabel.SetText(item.ErrorMessage)
	}

	if item.CanResume() {
		d.pauseBtn.SetText("Resume")
		d.pauseBtn.SetIconSource(ui.IconPath("MdiLightPlay.svg"))
	} else {
		d.pauseBtn.SetText("Pause")
		d.pauseBtn.SetIconSource(ui.IconPath("MdiLightPause.svg"))
	}
	d.pauseBtn.SetEnabled(item.CanPause() || item.CanResume())
	d.stopBtn.SetEnabled(item.CanPause())
}

func (d *DownloadItemDetail) SetSamples(samples []int64) {
	if d == nil {
		return
	}
	d.chart.SetSamples(samples)
}

func (d *DownloadItemDetail) OnPause(fn func(string)) {
	if fn != nil {
		d.pauseCB = append(d.pauseCB, fn)
	}
}

func (d *DownloadItemDetail) OnResume(fn func(string)) {
	if fn != nil {
		d.resumeCB = append(d.resumeCB, fn)
	}
}

func (d *DownloadItemDetail) OnStop(fn func(string)) {
	if fn != nil {
		d.stopCB = append(d.stopCB, fn)
	}
}

func (d *DownloadItemDetail) OnRemove(fn func(string, bool)) {
	if fn != nil {
		d.removeCB = append(d.removeCB, fn)
	}
}

func categoryIcon(category lib.Category) string {
	switch category {
	case lib.CategoryAudio:
		return "MdiLightMusic.svg"
	case lib.CategoryApplication:
		return "MdiLightConsole.svg"
	case lib.CategoryCompressed:
		return "MdiLightViewModule.svg"
	case lib.CategoryDocument:
		return "MdiLightBook.svg"
	case lib.CategoryImage:
		return "MdiLightPicture.svg"
	case lib.CategoryVideo:
		return "MdiLightFilmstrip.svg"
	default:
		return "MdiLightHelpCircle.svg"
	}
}
