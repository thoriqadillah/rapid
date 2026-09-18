package components

import (
	"fmt"

	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

// SidebarItem is one selectable navigation row. The parent owns routing;
// this widget only owns its destination data and activation callbacks.
type SidebarItem struct {
	*qt.QWidget

	iconLabel  *qt.QLabel
	label      *qt.QLabel
	countLabel *qt.QLabel

	destination string
	iconSource  string
	iconColor   *qt.QColor
	category    bool
	selected    bool
	hovered     bool
	callbacks   []func()
}

func NewSidebarItem(destination, label string) *SidebarItem {
	i := &SidebarItem{
		QWidget:     qt.NewQWidget3(nil, 0),
		destination: destination,
		iconColor:   theme.ColorTextMuted,
	}
	i.SetObjectName(*qt.NewQAnyStringView3("sidebarItem"))
	i.SetAttribute(qt.WA_StyledBackground)
	i.SetAttribute(qt.WA_Hover)
	i.SetMinimumHeight(36)
	i.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
	i.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	i.SetMouseTracking(true)

	i.iconLabel = qt.NewQLabel3("")
	i.iconLabel.SetStyleSheet("QLabel { background: transparent; border: none; outline: none; }")
	i.iconLabel.SetAlignment(qt.AlignCenter)
	i.iconLabel.SetFixedSize2(theme.IconSm, theme.IconSm)
	i.iconLabel.SetAttribute(qt.WA_TransparentForMouseEvents)

	i.label = qt.NewQLabel3(label)
	i.label.SetStyleSheet("QLabel { background: transparent; border: none; outline: none; }")
	i.label.SetAlignment(qt.AlignLeft | qt.AlignVCenter)
	i.label.SetWordWrap(false)
	i.label.SetMinimumWidth(0)
	i.label.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Preferred)
	i.label.SetAttribute(qt.WA_TransparentForMouseEvents)

	i.countLabel = qt.NewQLabel3("")
	i.countLabel.SetStyleSheet("QLabel { background: transparent; border: none; outline: none; }")
	i.countLabel.SetAlignment(qt.AlignRight | qt.AlignVCenter)
	i.countLabel.SetSizePolicy2(qt.QSizePolicy__Maximum, qt.QSizePolicy__Preferred)
	i.countLabel.SetAttribute(qt.WA_TransparentForMouseEvents)
	i.countLabel.SetVisible(false)

	row := qt.NewQHBoxLayout2()
	row.SetContentsMargins(theme.SpacingSm, 0, theme.SpacingSm, 0)
	row.SetSpacing(theme.SpacingSm)
	row.AddWidget(i.iconLabel.QWidget)
	row.AddWidget(i.label.QWidget)
	row.AddWidget(i.countLabel.QWidget)
	i.SetLayout(row.QLayout)

	i.OnEnterEvent(func(super func(*qt.QEnterEvent), event *qt.QEnterEvent) {
		super(event)
		i.hovered = true
		i.refreshStyle()
		i.Update()
	})
	i.OnLeaveEvent(func(super func(*qt.QEvent), event *qt.QEvent) {
		super(event)
		i.hovered = false
		i.refreshStyle()
		i.Update()
	})
	i.OnMousePressEvent(func(super func(*qt.QMouseEvent), event *qt.QMouseEvent) {
		super(event)
		i.activate()
	})
	i.OnPaintEvent(func(super func(*qt.QPaintEvent), event *qt.QPaintEvent) {
		super(event)
		i.paintBackground()
	})

	i.refreshStyle()
	return i
}

func (i *SidebarItem) SetDestination(v string) {
	i.destination = v
}

func (i *SidebarItem) Destination() string {
	return i.destination
}

func (i *SidebarItem) SetLabel(v string) {
	i.label.SetText(v)
	i.refreshStyle()
}

func (i *SidebarItem) Label() string {
	return i.label.Text()
}

func (i *SidebarItem) SetCount(v string) {
	i.countLabel.SetText(v)
	i.countLabel.SetVisible(v != "")
	i.refreshStyle()
}

func (i *SidebarItem) Count() string {
	return i.countLabel.Text()
}

func (i *SidebarItem) CountVisible() bool {
	return i.countLabel.Text() != ""
}

func (i *SidebarItem) SetIconSource(v string) {
	i.iconSource = v
	i.refreshIcon()
}

func (i *SidebarItem) IconSource() string {
	return i.iconSource
}

func (i *SidebarItem) SetIconColor(v *qt.QColor) {
	if v == nil {
		v = theme.ColorTextMuted
	}
	i.iconColor = v
	i.refreshIcon()
}

func (i *SidebarItem) IconColor() *qt.QColor {
	return i.iconColor
}

func (i *SidebarItem) SetCategoryItem(v bool) {
	i.category = v
	size := theme.IconSm
	if v {
		size = theme.IconXs
	}
	i.iconLabel.SetFixedSize2(size, size)
	i.refreshIcon()
}

func (i *SidebarItem) CategoryItem() bool {
	return i.category
}

func (i *SidebarItem) SetSelected(v bool) {
	i.selected = v
	i.refreshStyle()
	i.refreshIcon()
	i.Update()
}

func (i *SidebarItem) Selected() bool {
	return i.selected
}

func (i *SidebarItem) OnActivated(fn func()) {
	if fn != nil {
		i.callbacks = append(i.callbacks, fn)
	}
}

func (i *SidebarItem) activate() {
	for _, fn := range i.callbacks {
		if fn != nil {
			fn()
		}
	}
}

func (i *SidebarItem) refreshIcon() {
	if i.iconSource == "" {
		i.iconLabel.SetPixmap(qt.NewQPixmap2(0, 0))
		return
	}
	color := i.iconColor
	if i.selected && !i.category {
		color = theme.ColorText
	}
	size := theme.IconSm
	if i.category {
		size = theme.IconXs
	}
	pixmap := ui.TintedPixmap(i.iconSource, color, size)
	if pixmap == nil {
		pixmap = qt.NewQPixmap2(0, 0)
	}
	i.iconLabel.SetPixmap(pixmap)
}

func (i *SidebarItem) refreshStyle() {
	foreground := theme.ColorTextMuted
	if i.selected || i.hovered {
		foreground = theme.ColorText
	}
	i.SetStyleSheet(fmt.Sprintf(`
		QWidget#sidebarItem {
			background: transparent;
			border: none;
			outline: none;
		}
		QWidget#sidebarItem QLabel {
			background: transparent;
			border: none;
			outline: none;
			color: %s;
		}
	`, theme.CssColor(foreground)))
}

// paintBackground is deliberate rather than stylesheet-only. On this MIQT
// backend, a plain QWidget can expose a stylesheet background without
// painting it; drawing in the row's own paint event makes the QML fill
// deterministic.
func (i *SidebarItem) backgroundColor() *qt.QColor {
	if !i.selected && !i.hovered || theme.ColorSurface == nil {
		return nil
	}
	return theme.ColorSurface.LighterWithInt(140)
}

func (i *SidebarItem) paintBackground() {
	background := i.backgroundColor()
	if background == nil {
		return
	}
	painter := qt.NewQPainter2(i.QPaintDevice)
	defer painter.End()
	painter.SetRenderHint2(qt.QPainter__Antialiasing, true)
	painter.SetPen(qt.NewQColor11(0, 0, 0, 0))
	painter.SetBrush(qt.NewQBrush3(background))
	painter.DrawRoundedRect2(0, 0, i.Width(), i.Height(), float64(theme.RadiusSm), float64(theme.RadiusSm))
}
