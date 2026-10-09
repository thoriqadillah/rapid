package components

import (
	"fmt"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

const (
	SidebarWidth        = 200
	sidebarAnimDuration = 200
)

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

// SetCounts updates badges keyed by destination; zero/absent hides the badge.
func (s *Sidebar) SetCounts(counts map[string]int) {
	for _, section := range s.sections {
		section.SetCounts(counts)
	}
}

func (s *Sidebar) CurrentDestination() string {
	return s.current
}

func (s *Sidebar) SetOpen(v bool) {
	if s.open == v {
		return
	}
	s.open = v
	target := 0
	if v {
		target = SidebarWidth
	}
	s.animate("minimumWidth", target)
	s.animate("maximumWidth", target)
}

func (s *Sidebar) animate(prop string, target int) {
	anim := qt.NewQPropertyAnimation2(qt.UnsafeNewQObject(s.UnsafePointer()), []byte(prop))
	anim.SetParent(qt.UnsafeNewQObject(s.UnsafePointer())) // Qt owns the animation
	anim.SetDuration(sidebarAnimDuration)
	start := qt.NewQVariant4(s.Width())
	end := qt.NewQVariant4(target)
	easingCurve := qt.NewQEasingCurve3(qt.QEasingCurve__InOutCubic)
	anim.SetStartValue(start)
	anim.SetEndValue(end)
	anim.SetEasingCurve(easingCurve)
	start.Delete()
	end.Delete()
	easingCurve.Delete()
	anim.Start()
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
		"background-color: %s;",
		theme.CssColor(theme.ColorSurface),
	))
}
