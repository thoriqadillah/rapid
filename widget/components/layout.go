package components

import (
	"rapid/lib/reactive"

	qt "github.com/mappu/miqt/qt6"
)

// Layout is the explicit Qt Widgets equivalent of the QML Loader/slot layout.
type Layout struct {
	*qt.QWidget
	headerWidget  *Header
	SidebarWidget *Sidebar
	contentWidget *qt.QWidget
	contentLayout *qt.QVBoxLayout

	grid        *qt.QGridLayout
	sidebarOpen bool
	addCB       []func()
}

func NewLayout() *Layout {
	l := &Layout{
		QWidget:       qt.NewQWidget3(nil, 0),
		contentWidget: qt.NewQWidget2(),
		contentLayout: qt.NewQVBoxLayout2(),
		grid:          qt.NewQGridLayout2(),
		sidebarOpen:   true,
	}
	l.contentLayout.SetContentsMargins(0, 0, 0, 0)
	l.contentWidget.SetLayout(l.contentLayout.QLayout)
	l.contentWidget.SetFocusPolicy(qt.StrongFocus)

	l.grid.SetContentsMargins(0, 0, 0, 0)
	l.grid.SetSpacing(0)
	l.grid.SetColumnStretch(0, 0)
	l.grid.SetColumnStretch(1, 1)
	l.grid.SetRowStretch(0, 0)
	l.grid.SetRowStretch(1, 1)
	l.SetLayout(l.grid.QLayout)
	l.grid.AddWidget3(l.contentWidget, 1, 1, 1, 1)
	l.SetSidebar(nil)
	l.SetHeader(nil)
	return l
}

func (l *Layout) SetHeader(header *Header) {
	if l.headerWidget != nil {
		l.grid.RemoveWidget(l.headerWidget.QWidget)
		l.headerWidget.Hide()
		l.headerWidget.DeleteLater()
	}
	if header == nil {
		header = NewHeader()
	}
	l.headerWidget = header
	l.grid.AddWidget2(header.QWidget, 0, 1)
	header.OnMenuClicked(func() { l.SetSidebarOpen(!l.sidebarOpen) })
	header.OnAddClicked(func() {
		for _, fn := range l.addCB {
			if fn != nil {
				fn()
			}
		}
	})
}

func (l *Layout) SetSidebar(sidebar *Sidebar) {
	if l.SidebarWidget != nil {
		l.grid.RemoveWidget(l.SidebarWidget.QWidget)
		l.SidebarWidget.Hide()
		l.SidebarWidget.DeleteLater()
	}
	if sidebar == nil {
		sidebar = NewSidebar()
	}
	l.SidebarWidget = sidebar
	l.sidebarOpen = sidebar.Open()
	l.grid.AddWidget3(sidebar.QWidget, 0, 0, 2, 1)
}

func (l *Layout) SetSidebarOpen(open bool) {
	l.sidebarOpen = open
	if l.SidebarWidget != nil {
		l.SidebarWidget.SetOpen(open)
	}
}

func (l *Layout) SidebarOpen() bool {
	return l.sidebarOpen
}

func (l *Layout) BindSearch(search reactive.Model[string]) {
	l.headerWidget.searchField.Bind(search)
}

func (l *Layout) BindDestination(destination reactive.Model[string]) {
	l.SidebarWidget.Bind(destination)
}

func (l *Layout) AddContentWidget(widget *qt.QWidget) {
	if widget != nil {
		l.contentLayout.AddWidget(widget)
	}
}

func (l *Layout) OnAddClicked(fn func()) {
	if fn != nil {
		l.addCB = append(l.addCB, fn)
	}
}

func (l *Layout) FocusContent() {
	if l.contentWidget != nil {
		l.contentWidget.SetFocus()
	}
}
