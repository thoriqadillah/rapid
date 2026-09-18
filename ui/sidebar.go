package ui

import (
	"fmt"

	"rapid/theme"

	qt "github.com/mappu/miqt/qt6"
)

const SidebarWidth = 200

// Sidebar is the left navigation rail. Width changes are immediate; this is a
// deliberate native-Qt deviation from QML's 200ms animation.
type Sidebar struct {
	*qt.QWidget
	ContentLayout *qt.QVBoxLayout

	current   string
	open      bool
	sections  []*SidebarSection
	callbacks []func(string)
}

func NewSidebar() *Sidebar {
	s := &Sidebar{
		QWidget:       qt.NewQWidget3(nil, 0),
		ContentLayout: qt.NewQVBoxLayout2(),
		open:          true,
	}
	s.SetMinimumWidth(SidebarWidth)
	s.SetMaximumWidth(SidebarWidth)
	s.SetSizePolicy2(qt.QSizePolicy__Fixed, qt.QSizePolicy__Expanding)
	s.SetAttribute(qt.WA_StyledBackground)
	s.ContentLayout.SetContentsMargins(theme.SpacingSm, theme.SpacingSm, theme.SpacingSm, theme.SpacingSm)
	s.ContentLayout.SetSpacing(theme.SpacingXs)
	s.SetLayout(s.ContentLayout.QLayout)
	s.refreshStyle()
	return s
}

func (s *Sidebar) SetCurrentDestination(v string) {
	s.current = v
	for _, section := range s.sections {
		section.SetCurrentDestination(v)
	}
}
func (s *Sidebar) CurrentDestination() string {
	return s.current
}

func (s *Sidebar) SetOpen(v bool) {
	s.open = v
	if v {
		s.SetMinimumWidth(SidebarWidth)
		s.SetMaximumWidth(SidebarWidth)
	} else {
		s.SetMinimumWidth(0)
		s.SetMaximumWidth(0)
	}
}

func (s *Sidebar) Open() bool {
	return s.open
}

func (s *Sidebar) Activate(destination string) {
	s.current = destination
	s.SetCurrentDestination(destination)
	for _, fn := range s.callbacks {
		if fn != nil {
			fn(destination)
		}
	}
}

func (s *Sidebar) AddContentWidget(widget *qt.QWidget) {
	if widget != nil {
		s.ContentLayout.AddWidget(widget)
	}
}

// AddStretch inserts flexible vertical space between navigation groups.
func (s *Sidebar) AddStretch() {
	s.ContentLayout.AddStretch()
}

func (s *Sidebar) AddSection(section *SidebarSection) {
	if section == nil {
		return
	}
	s.sections = append(s.sections, section)
	section.SetCurrentDestination(s.current)
	section.OnActivated(s.Activate)
	s.AddContentWidget(section.QWidget)
}

func (s *Sidebar) OnDestinationSelected(fn func(string)) {
	if fn != nil {
		s.callbacks = append(s.callbacks, fn)
	}
}

func (s *Sidebar) refreshStyle() {
	s.SetStyleSheet(fmt.Sprintf(
		"background-color: %s; border-right: 1px solid %s;",
		theme.CssColor(theme.ColorSurface), theme.CssColor(theme.ColorBorder),
	))
}
