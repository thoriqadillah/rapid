package downloads

import (
	"rapid/lib/helpers/bools"
	"rapid/lib/reactive"
	"rapid/services/download/api"
	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

// DownloadList owns the rows, single-expand state, context-menu highlight and
// the empty state. It rebuilds by diffing gids so only added/removed rows
// animate.
type DownloadList struct {
	*qt.QWidget

	scroll          *qt.QScrollArea
	container       *qt.QWidget
	containerLayout *qt.QVBoxLayout
	empty           *qt.QWidget
	menu            *ItemContextMenu

	items []*downloadItem
	byGid map[string]*downloadItem

	expandedGid string
	emptyShown  bool
	emptySet    bool

	pauseCB         []func(gid string)
	resumeCB        []func(gid string)
	stopCB          []func(gid string)
	removeCB        []func(gid string, deleteFromDisk bool)
	deleteRequestCB []func(api.Download)
	expandCB        []func(gid string)
}

func NewDownloadList(parent *qt.QWidget) *DownloadList {
	l := &DownloadList{
		QWidget: qt.NewQWidget3(parent, 0),
		byGid:   make(map[string]*downloadItem),
	}
	l.menu = NewItemContextMenu(l.QWidget)
	l.menu.OnPause(func(gid string) { l.emitGid(l.pauseCB, gid) })
	l.menu.OnResume(func(gid string) { l.emitGid(l.resumeCB, gid) })
	l.menu.OnStop(func(gid string) { l.emitGid(l.stopCB, gid) })
	l.menu.OnDelete(func(d api.Download) {
		l.clearMenuState()
		for _, fn := range l.deleteRequestCB {
			if fn != nil {
				fn(d)
			}
		}
	})
	l.menu.OnClosed(l.clearMenuState)

	l.container = qt.NewQWidget3(nil, 0)
	l.containerLayout = qt.NewQVBoxLayout2()
	l.containerLayout.SetContentsMargins(0, 0, 0, 0)
	l.containerLayout.SetSpacing(0)
	l.containerLayout.AddStretch()
	l.container.SetLayout(l.containerLayout.QLayout)

	l.scroll = qt.NewQScrollArea(l.QWidget)
	l.scroll.SetWidgetResizable(true)
	l.scroll.SetFrameShape(qt.QFrame__NoFrame)
	l.scroll.SetHorizontalScrollBarPolicy(qt.ScrollBarAlwaysOff)
	l.scroll.SetVerticalScrollBarPolicy(qt.ScrollBarAsNeeded)
	l.scroll.SetStyleSheet(`
		QScrollArea {
			border: none;
			background: transparent;
		}
		QScrollArea QWidget#qt_scrollarea_viewport {
			background: transparent;
		}
	`)
	l.scroll.SetWidget(l.container)

	l.empty = newEmptyState(l.QWidget)

	layout := qt.NewQVBoxLayout2()
	layout.SetContentsMargins(0, 0, 0, 0)
	layout.SetSpacing(0)
	layout.AddWidget2(l.scroll.QWidget, 1)
	layout.AddWidget2(l.empty, 1)
	l.SetLayout(layout.QLayout)
	l.updateEmpty()
	return l
}

func newEmptyState(parent *qt.QWidget) *qt.QWidget {
	w := qt.NewQWidget3(parent, 0)
	icon := qt.NewQLabel3("")
	icon.SetFixedSize2(theme.IconXl, theme.IconXl)
	icon.SetPixmap(ui.TintedPixmap(ui.IconPath("MdiLightFormatAlignBottom.svg"), theme.ColorTextMuted, theme.IconXl))
	icon.SetAlignment(qt.AlignCenter)

	text := styledLabel("No downloads yet", theme.ColorTextMuted)
	text.SetAlignment(qt.AlignCenter)

	column := qt.NewQVBoxLayout2()
	column.SetContentsMargins(0, 0, 0, 0)
	column.SetSpacing(theme.SpacingMd)
	column.AddStretch()
	column.AddWidget3(icon.QWidget, 0, qt.AlignHCenter)
	column.AddWidget3(text.QWidget, 0, qt.AlignHCenter)
	column.AddStretch()
	w.SetLayout(column.QLayout)
	return w
}

func (l *DownloadList) BindItems(items *reactive.Computed[[]api.Download]) {
	l.OnDestroyed(reactive.Effect(func() {
		l.updateItems(items.Get())
	}))
}

func (l *DownloadList) BindFilterChange(isChanged *reactive.Computed[bool]) {
	l.OnDestroyed(reactive.Effect(func() {
		isChanged.Get()
		l.collapse()
	}))
}

// updateItems refreshes the list. Insertion order is preserved (the store already
// sorts newest-first); the list never re-sorts.
func (l *DownloadList) updateItems(items []api.Download) {
	wanted := make(map[string]bool, len(items))
	ordered := make([]*downloadItem, 0, len(items))
	for i, d := range items {
		wanted[d.GID] = true
		it := l.byGid[d.GID]
		if it == nil {
			it = newDownloadItem(l.container)
			l.wireItem(it)
			l.animateIn(it)
		}
		it.SetIndex(i)
		it.SetItem(d)
		it.SetExpanded(d.GID == l.expandedGid)
		ordered = append(ordered, it)
	}

	for gid, it := range l.byGid {
		if !wanted[gid] {
			l.animateOut(it)
			delete(l.byGid, gid)
		}
	}

	if !sameOrder(l.items, ordered) {
		l.detachAll()
		l.byGid = make(map[string]*downloadItem, len(ordered))
		for _, it := range ordered {
			l.containerLayout.AddWidget(it.QWidget)
			l.byGid[it.item.GID] = it
		}
		l.containerLayout.AddStretch()
		l.items = ordered
	}

	if l.expandedGid != "" && !wanted[l.expandedGid] {
		l.expandedGid = ""
		l.emitExpand()
	}
	l.updateEmpty()
}

func sameOrder(a, b []*downloadItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].item.GID != b[i].item.GID {
			return false
		}
	}
	return true
}

// detachAll removes every widget (and the trailing stretch) from the layout.
// RemoveWidget owns and frees the layout item, so no wrapper leaks.
func (l *DownloadList) detachAll() {
	for _, it := range l.items {
		l.containerLayout.RemoveWidget(it.QWidget)
	}
	for l.containerLayout.Count() > 0 {
		if item := l.containerLayout.TakeAt(0); item != nil {
			item.Delete()
		}
	}
}

func (l *DownloadList) wireItem(it *downloadItem) {
	it.OnToggle(func() { l.toggle(it.item.GID) })
	it.OnContextMenu(func(d api.Download, pos *qt.QPoint) { l.openMenu(d, pos) })
	it.OnPause(func(gid string) { l.emitGid(l.pauseCB, gid) })
	it.OnResume(func(gid string) { l.emitGid(l.resumeCB, gid) })
	it.OnStop(func(gid string) { l.emitGid(l.stopCB, gid) })
	it.OnRemove(func(gid string, deleteFromDisk bool) {
		for _, fn := range l.removeCB {
			if fn != nil {
				fn(gid, deleteFromDisk)
			}
		}
	})
}

func (l *DownloadList) toggle(gid string) {
	l.expandedGid = bools.Ternary(l.expandedGid == gid, "", gid)
	for _, it := range l.items {
		it.SetExpanded(it.item.GID == l.expandedGid)
	}
	l.emitExpand()
}

// collapse closes the expanded row (category/search changes do this).
func (l *DownloadList) collapse() {
	if l.expandedGid == "" {
		return
	}
	l.expandedGid = ""
	for _, it := range l.items {
		it.SetExpanded(false)
	}
	l.emitExpand()
}

// SetSpeedHistory feeds the sparkline of the matching row.
func (l *DownloadList) SetSpeedHistory(gid string, samples []int64) {
	if it, ok := l.byGid[gid]; ok && it != nil {
		it.SetSamples(samples)
	}
}

func (l *DownloadList) openMenu(d api.Download, pos *qt.QPoint) {
	for _, it := range l.items {
		it.SetMenuOpen(true)
	}
	if it := l.byGid[d.GID]; it != nil {
		it.SetHighlighted(true)
	}
	l.menu.ShowFor(d, pos)
}

func (l *DownloadList) clearMenuState() {
	for _, it := range l.items {
		it.SetMenuOpen(false)
		it.SetHighlighted(false)
	}
}

func (l *DownloadList) updateEmpty() {
	empty := len(l.items) == 0
	if l.emptySet && l.emptyShown == empty {
		return // SetVisible to the same value still touches the layout
	}
	l.emptySet = true
	l.emptyShown = empty
	l.empty.SetVisible(empty)
	l.scroll.SetVisible(!empty)
}

func (l *DownloadList) animateIn(it *downloadItem) {
	effect := qt.NewQGraphicsOpacityEffect2(it.QObject)
	effect.SetOpacity(0)
	it.SetGraphicsEffect(effect.QGraphicsEffect)
	animateOpacity(effect, 0, 1, addRemoveDuration, qt.QEasingCurve__OutCubic, func() {
		it.SetGraphicsEffect(nil)
	})
	target := max(it.SizeHint().Height(), theme.TouchTarget)
	animateMaxHeight(it.QWidget, 0, target, addRemoveDuration, qt.QEasingCurve__OutCubic, func() {
		it.SetMaximumHeight(maxWidgetHeight)
	})
}

func (l *DownloadList) animateOut(it *downloadItem) {
	g := it.Geometry()
	effect := qt.NewQGraphicsOpacityEffect2(it.QObject)
	effect.SetOpacity(1)
	it.SetGraphicsEffect(effect.QGraphicsEffect)
	animateOpacity(effect, 1, 0, addRemoveDuration, qt.QEasingCurve__InCubic, func() {
		it.Hide()
		it.DeleteLater()
	})
	// Float the fading row at its old spot while the rest reflows.
	it.SetGeometry(g.X(), g.Y(), g.Width(), g.Height())
	it.Raise()
}

func (l *DownloadList) emitGid(callbacks []func(string), gid string) {
	for _, fn := range callbacks {
		if fn != nil {
			fn(gid)
		}
	}
}

func (l *DownloadList) emitExpand() {
	for _, fn := range l.expandCB {
		if fn != nil {
			fn(l.expandedGid)
		}
	}
}

func (l *DownloadList) OnPause(fn func(gid string)) {
	if fn != nil {
		l.pauseCB = append(l.pauseCB, fn)
	}
}

func (l *DownloadList) OnResume(fn func(gid string)) {
	if fn != nil {
		l.resumeCB = append(l.resumeCB, fn)
	}
}

func (l *DownloadList) OnStop(fn func(gid string)) {
	if fn != nil {
		l.stopCB = append(l.stopCB, fn)
	}
}

func (l *DownloadList) OnRemove(fn func(gid string, deleteFromDisk bool)) {
	if fn != nil {
		l.removeCB = append(l.removeCB, fn)
	}
}

func (l *DownloadList) OnDeleteRequested(fn func(api.Download)) {
	if fn != nil {
		l.deleteRequestCB = append(l.deleteRequestCB, fn)
	}
}

// OnExpand fires with the expanded gid ("" after collapsing).
func (l *DownloadList) OnExpand(fn func(gid string)) {
	if fn != nil {
		l.expandCB = append(l.expandCB, fn)
	}
}
