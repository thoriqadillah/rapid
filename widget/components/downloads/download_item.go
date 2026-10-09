package downloads

import (
	"reflect"

	"rapid/services/download/api"
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

const maxWidgetHeight = 16777215

// downloadItem is one row: category dot, name, progress bar + %, speed, size,
// ETA, plus the expandable detail.
type downloadItem struct {
	*qt.QWidget

	item        api.Download
	index       int
	highlighted bool
	menuOpen    bool
	hovered     bool

	dot          *qt.QLabel
	name         *elidedLabel
	progress     *ProgressBar
	progressText *qt.QLabel
	speed        *qt.QLabel
	size         *qt.QLabel
	eta          *qt.QLabel

	detail         *DownloadItemDetail
	detailBox      *qt.QWidget
	detailEffect   *qt.QGraphicsOpacityEffect
	detailExpanded bool
	heightAnim     *qt.QPropertyAnimation
	opacityAnim    *qt.QPropertyAnimation

	toggleCB  []func()
	contextCB []func(api.Download, *qt.QPoint)
	pauseCB   []func(string)
	resumeCB  []func(string)
	stopCB    []func(string)
	removeCB  []func(string, bool)
}

func newDownloadItem(parent *qt.QWidget) *downloadItem {
	it := &downloadItem{QWidget: qt.NewQWidget3(parent, 0)}
	it.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Preferred)
	it.SetMinimumHeight(theme.TouchTarget + 2*theme.SpacingSm)
	it.SetAttribute(qt.WA_Hover)
	it.SetMouseTracking(true)
	it.SetFocusPolicy(qt.StrongFocus)
	// SetCursor copies the cursor, so the temporary can be released at once.
	cursor := qt.NewQCursor2(qt.PointingHandCursor)
	it.SetCursor(cursor)
	cursor.Delete()
	it.SetContextMenuPolicy(qt.CustomContextMenu)

	it.dot = qt.NewQLabel3("")
	it.dot.SetFixedSize2(theme.IconXs, theme.IconXs)
	it.dot.SetAlignment(qt.AlignCenter)
	it.dot.SetAttribute(qt.WA_TransparentForMouseEvents)

	it.name = newElidedLabel("", theme.ColorText, qt.ElideRight)
	it.name.SetMinimumWidth(NameColumnMinWidth)

	it.progress = NewProgressBar(it.QWidget)
	it.progressText = styledLabel("0%", theme.ColorTextMuted)
	it.progressText.SetAlignment(qt.AlignLeft | qt.AlignVCenter)
	progressFont := it.progressText.Font()
	progressFont.SetPixelSize(theme.TextSize)
	it.progressText.SetFont(progressFont)
	progressMetrics := qt.NewQFontMetrics(progressFont)
	it.progressText.SetFixedWidth(progressMetrics.HorizontalAdvance("100%"))
	progressMetrics.Delete()

	progressBox := qt.NewQWidget3(it.QWidget, 0)
	progressBox.SetFixedWidth(ProgressColumnWidth)
	progressBox.SetAttribute(qt.WA_TransparentForMouseEvents)
	progressRow := qt.NewQHBoxLayout2()
	progressRow.SetContentsMargins(0, 0, 0, 0)
	progressRow.SetSpacing(theme.SpacingSm)
	progressRow.AddWidget2(it.progress.QWidget, 1)
	progressRow.AddWidget(it.progressText.QWidget)
	progressBox.SetLayout(progressRow.QLayout)

	it.speed = fixedLabel("—", theme.ColorText, SpeedColumnWidth)
	it.size = fixedLabel("—", theme.ColorText, SizeColumnWidth)
	it.eta = fixedLabel("—", theme.ColorText, EtaColumnWidth)

	row := qt.NewQHBoxLayout2()
	row.SetContentsMargins(theme.SpacingLg, theme.SpacingSm, theme.SpacingLg, theme.SpacingSm)
	row.SetSpacing(theme.SpacingMd)
	row.AddWidget(it.dot.QWidget)
	row.AddWidget2(it.name.QWidget, 1)
	row.AddWidget(progressBox)
	row.AddWidget(it.speed.QWidget)
	row.AddWidget(it.size.QWidget)
	row.AddWidget(it.eta.QWidget)

	it.detailBox = qt.NewQWidget3(it.QWidget, 0)
	detailBoxLayout := qt.NewQVBoxLayout2()
	detailBoxLayout.SetContentsMargins(0, 0, 0, 0)
	detailBoxLayout.SetSpacing(0)
	it.detail = NewDownloadItemDetail(it.detailBox)
	detailBoxLayout.AddWidget(it.detail.QWidget)
	it.detailBox.SetLayout(detailBoxLayout.QLayout)
	it.detailBox.SetMaximumHeight(0)
	it.detailBox.Hide()

	outer := qt.NewQVBoxLayout2()
	outer.SetContentsMargins(0, 0, 0, 0)
	outer.SetSpacing(0)
	outer.AddLayout(row.QLayout)
	outer.AddWidget(it.detailBox)
	it.SetLayout(outer.QLayout)

	it.detail.OnPause(func(gid string) { it.emit(it.pauseCB, gid) })
	it.detail.OnResume(func(gid string) { it.emit(it.resumeCB, gid) })
	it.detail.OnStop(func(gid string) { it.emit(it.stopCB, gid) })
	it.detail.OnRemove(func(gid string, deleteFromDisk bool) {
		for _, fn := range it.removeCB {
			if fn != nil {
				fn(gid, deleteFromDisk)
			}
		}
	})

	it.OnEnterEvent(func(super func(*qt.QEnterEvent), event *qt.QEnterEvent) {
		super(event)
		it.hovered = true
		it.Update()
	})
	it.OnLeaveEvent(func(super func(*qt.QEvent), event *qt.QEvent) {
		super(event)
		it.hovered = false
		it.Update()
	})
	it.OnMousePressEvent(func(super func(*qt.QMouseEvent), event *qt.QMouseEvent) {
		super(event)
		if event.Button() == qt.LeftButton {
			it.toggle()
		}
	})
	it.OnKeyPressEvent(func(super func(event *qt.QKeyEvent), event *qt.QKeyEvent) {
		switch qt.Key(event.Key()) {
		case qt.Key_Return, qt.Key_Enter, qt.Key_Space:
			it.toggle()
			return
		}
		super(event)
	})
	it.OnCustomContextMenuRequested(func(pos *qt.QPoint) {
		global := it.MapToGlobalWithQPoint(pos)
		for _, fn := range it.contextCB {
			if fn != nil {
				fn(it.item, global)
			}
		}
	})
	it.OnPaintEvent(func(super func(*qt.QPaintEvent), event *qt.QPaintEvent) {
		super(event)
		it.paint()
	})

	return it
}

func (it *downloadItem) toggle() {
	for _, fn := range it.toggleCB {
		if fn != nil {
			fn()
		}
	}
}

func (it *downloadItem) emit(callbacks []func(string), gid string) {
	for _, fn := range callbacks {
		if fn != nil {
			fn(gid)
		}
	}
}

// SetItem updates every row field from the download.
//
// This is called for every row on every list refresh. When a live download
// ticks, only one row actually changes; skip the rest entirely so an unchanged
// row does no SetText/SetPixmap/repaint work (and no native Qt churn).
func (it *downloadItem) SetItem(d api.Download) {
	if reflect.DeepEqual(it.item, d) {
		return
	}
	it.item = d
	it.SetAccessibleName(d.Name())
	color := theme.CategoryColor(d.Category)
	it.dot.SetPixmap(ui.TintedPixmap(ui.IconPath("MdiSquareRounded.svg"), color, theme.IconXs))
	it.name.SetFullText(d.Name())
	it.progress.SetValue(min(1, d.Progress()), color, d.IsError())
	it.progressText.SetText(d.FormatProgress())
	it.speed.SetText(d.FormatSpeed())
	it.size.SetText(d.FormatSize())
	it.eta.SetText(d.FormatEstimation())
	it.detail.SetItem(d)
}

func (it *downloadItem) Item() api.Download {
	return it.item
}

func (it *downloadItem) SetIndex(index int) {
	if it.index == index {
		return
	}
	it.index = index
	it.Update()
}

func (it *downloadItem) SetHighlighted(on bool) {
	if it.highlighted == on {
		return
	}
	it.highlighted = on
	it.Update()
}

func (it *downloadItem) Highlighted() bool {
	return it.highlighted
}

func (it *downloadItem) SetMenuOpen(on bool) {
	if it.menuOpen == on {
		return
	}
	it.menuOpen = on
	it.Update()
}

func (it *downloadItem) SetExpanded(on bool) {
	if it.detailExpanded == on {
		return
	}
	it.detailExpanded = on
	it.Update()
	it.stopDetailAnims()
	if on {
		it.detail.SetMinimumHeight(0)
		it.detail.SetMaximumHeight(maxWidgetHeight)
		it.detailBox.Show()
		target := max(it.detail.SizeHint().Height(), theme.TouchTarget)
		it.detail.SetMinimumHeight(target)
		it.detail.SetMaximumHeight(target)
		it.heightAnim = animateMaxHeight(it.detailBox, 0, target, detailHeightDuration, qt.QEasingCurve__OutCubic, func() {
			it.detailBox.SetMaximumHeight(maxWidgetHeight)
			it.detail.SetMinimumHeight(0)
			it.detail.SetMaximumHeight(maxWidgetHeight)
		})
		it.ensureDetailEffect()
		it.detailEffect.SetOpacity(0)
		it.opacityAnim = animateOpacity(it.detailEffect, 0, 1, detailFadeDuration, qt.QEasingCurve__OutCubic, func() {
			it.detachDetailEffect()
		})
		return
	}
	// Pin the content to its full height so the shrinking box clips from the
	// bottom instead of squeezing.
	if full := it.detail.Height(); full > 0 {
		it.detail.SetMinimumHeight(full)
		it.detail.SetMaximumHeight(full)
	}
	from := it.detailBox.Height()
	it.heightAnim = animateMaxHeight(it.detailBox, from, 0, detailHeightDuration, qt.QEasingCurve__OutCubic, func() {
		it.detailBox.Hide()
	})
	it.ensureDetailEffect()
	it.detailEffect.SetOpacity(1)
	it.opacityAnim = animateOpacity(it.detailEffect, 1, 0, detailFadeDuration, qt.QEasingCurve__OutCubic, func() {
		it.detachDetailEffect()
	})
}

func (it *downloadItem) stopDetailAnims() {
	// Stop AND release the previous animations. They are parented to the row, so
	// leaving stopped instances around accumulated one pair per expand/collapse.
	if it.heightAnim != nil {
		it.heightAnim.Stop()
		it.heightAnim.DeleteLater()
		it.heightAnim = nil
	}
	if it.opacityAnim != nil {
		it.opacityAnim.Stop()
		it.opacityAnim.DeleteLater()
		it.opacityAnim = nil
	}
}

// ensureDetailEffect creates and installs the fade effect for one expand or
// collapse cycle. A stopped mid-fade effect stays installed and is reused;
// a detached one was deleted by Qt (see detachDetailEffect) and is remade.
func (it *downloadItem) ensureDetailEffect() {
	if it.detailEffect == nil {
		it.detailEffect = qt.NewQGraphicsOpacityEffect2(it.detailBox.QObject)
		it.detailBox.SetGraphicsEffect(it.detailEffect.QGraphicsEffect)
	} else if it.detailBox.GraphicsEffect() == nil {
		it.detailBox.SetGraphicsEffect(it.detailEffect.QGraphicsEffect)
	}
}

// detachDetailEffect removes the fade effect once its animation finished.
// Only finished (never stopped) anims reach here: Stop() emits no finished
// signal, so this cannot race stopDetailAnims. Detaching deletes the effect
// plus its finished child anim, hence both fields are nilled.
func (it *downloadItem) detachDetailEffect() {
	it.detailBox.SetGraphicsEffect(nil)
	it.detailEffect = nil
	it.opacityAnim = nil
}

func (it *downloadItem) Expanded() bool {
	return it.detailExpanded
}

func (it *downloadItem) SetSamples(samples []int64) {
	it.detail.SetSamples(samples)
}

func (it *downloadItem) OnToggle(fn func()) {
	if fn != nil {
		it.toggleCB = append(it.toggleCB, fn)
	}
}

func (it *downloadItem) OnContextMenu(fn func(api.Download, *qt.QPoint)) {
	if fn != nil {
		it.contextCB = append(it.contextCB, fn)
	}
}

func (it *downloadItem) OnPause(fn func(string)) {
	if fn != nil {
		it.pauseCB = append(it.pauseCB, fn)
	}
}

func (it *downloadItem) OnResume(fn func(string)) {
	if fn != nil {
		it.resumeCB = append(it.resumeCB, fn)
	}
}

func (it *downloadItem) OnStop(fn func(string)) {
	if fn != nil {
		it.stopCB = append(it.stopCB, fn)
	}
}

func (it *downloadItem) OnRemove(fn func(string, bool)) {
	if fn != nil {
		it.removeCB = append(it.removeCB, fn)
	}
}

// background: highlighted/expanded/hovered win over the odd-row zebra.
func (it *downloadItem) background() *qt.QColor {
	if it.highlighted || it.detailExpanded || (it.hovered && !it.menuOpen) {
		return theme.ColorSurface
	}
	if it.index%2 != 0 {
		return theme.WithAlpha(theme.ColorSurface, 64)
	}
	return nil
}

func (it *downloadItem) paint() {
	w, h := it.Width(), it.Height()
	if w <= 0 || h <= 0 {
		return
	}
	painter := qt.NewQPainter2(it.QPaintDevice)
	defer painter.Delete()
	if bg := it.background(); bg != nil {
		brush := qt.NewQBrush3(bg)
		painter.FillRect2(0, 0, w, h, brush)
		brush.Delete()
	}
	pen := qt.NewQPen3(theme.WithAlpha(theme.ColorBorder, 102))
	painter.SetPenWithPen(pen)
	painter.DrawLine2(0, h-1, w, h-1)
	pen.Delete()

	if it.HasFocus() {
		focusPen := qt.NewQPen3(theme.ColorPrimary)
		painter.SetPenWithPen(focusPen)
		painter.DrawRect2(0, 0, w-1, h-1)
		focusPen.Delete()
	}
}
