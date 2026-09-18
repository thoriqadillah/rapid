package components

import qt "github.com/mappu/miqt/qt6"

// Layout is the explicit Qt Widgets equivalent of the QML Loader/slot layout.
type Layout struct {
	*qt.QWidget
	HeaderWidget  *Header
	SidebarWidget *Sidebar
	ContentWidget *qt.QWidget
	ContentLayout *qt.QVBoxLayout

	grid          *qt.QGridLayout
	sidebarOpen   bool
	destinationCB []func(string)
	addCB         []func()
}

func NewLayout() *Layout {
	l := &Layout{
		QWidget:       qt.NewQWidget3(nil, 0),
		ContentWidget: qt.NewQWidget2(),
		ContentLayout: qt.NewQVBoxLayout2(),
		grid:          qt.NewQGridLayout2(),
		sidebarOpen:   true,
	}
	l.ContentLayout.SetContentsMargins(0, 0, 0, 0)
	l.ContentWidget.SetLayout(l.ContentLayout.QLayout)
	l.ContentWidget.SetFocusPolicy(qt.StrongFocus)

	l.grid.SetContentsMargins(0, 0, 0, 0)
	l.grid.SetSpacing(0)
	l.grid.SetColumnStretch(0, 0)
	l.grid.SetColumnStretch(1, 1)
	l.grid.SetRowStretch(0, 0)
	l.grid.SetRowStretch(1, 1)
	l.SetLayout(l.grid.QLayout)
	l.grid.AddWidget3(l.ContentWidget, 1, 1, 1, 1)
	l.SetSidebar(nil)
	l.SetHeader(nil)
	return l
}

func (l *Layout) SetHeader(header *Header) {
	if l.HeaderWidget != nil {
		l.grid.RemoveWidget(l.HeaderWidget.QWidget)
		l.HeaderWidget.Hide()
	}
	if header == nil {
		header = NewHeader()
	}
	l.HeaderWidget = header
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
	}
	if sidebar == nil {
		sidebar = NewSidebar()
	}
	l.SidebarWidget = sidebar
	l.sidebarOpen = sidebar.Open()
	l.grid.AddWidget3(sidebar.QWidget, 0, 0, 2, 1)
	sidebar.OnDestinationSelected(func(destination string) {
		for _, fn := range l.destinationCB {
			if fn != nil {
				fn(destination)
			}
		}
	})
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

func (l *Layout) AddContentWidget(widget *qt.QWidget) {
	if widget != nil {
		l.ContentLayout.AddWidget(widget)
	}
}

func (l *Layout) OnDestinationSelected(fn func(string)) {
	if fn != nil {
		l.destinationCB = append(l.destinationCB, fn)
	}
}

func (l *Layout) OnAddClicked(fn func()) {
	if fn != nil {
		l.addCB = append(l.addCB, fn)
	}
}

func (l *Layout) SearchText() string {
	if l.HeaderWidget == nil {
		return ""
	}
	return l.HeaderWidget.SearchText()
}

func (l *Layout) FocusContent() {
	if l.ContentWidget != nil {
		l.ContentWidget.SetFocus()
	}
}
