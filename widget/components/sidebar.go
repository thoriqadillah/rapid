package components

import (
	"fmt"

	"rapid/lib/reactive"
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

const (
	SidebarWidth        = 200
	sidebarAnimDuration = 200
)

type Sidebar struct {
	*qt.QWidget
	contentLayout *qt.QVBoxLayout

	destination *reactive.Signal[string]
	open        bool
	sections    []*SidebarSection
	callbacks   []func(string)
}

func NewSidebar() *Sidebar {
	s := &Sidebar{
		QWidget:       qt.NewQWidget3(nil, 0),
		contentLayout: qt.NewQVBoxLayout2(),
		open:          true,
		destination:   reactive.NewSignal(""),
	}
	s.SetMinimumWidth(SidebarWidth)
	s.SetMaximumWidth(SidebarWidth)
	s.SetSizePolicy2(qt.QSizePolicy__Fixed, qt.QSizePolicy__Expanding)
	s.SetAttribute(qt.WA_StyledBackground)
	s.contentLayout.SetContentsMargins(theme.SpacingSm, theme.SpacingSm, theme.SpacingSm, theme.SpacingSm)
	s.contentLayout.SetSpacing(theme.SpacingXs)
	s.SetLayout(s.contentLayout.QLayout)
	s.refreshStyle()
	return s
}

func (s *Sidebar) SetActive(dest string) {
	if dest == "" {
		return
	}

	s.destination.Set(dest)
	for _, section := range s.sections {
		section.SetActive(dest)
	}
}

func (s *Sidebar) Bind(dest reactive.Model[string]) {
	modelValue, dispose := reactive.Bind(dest)
	s.destination = modelValue
	s.OnDestroyed(dispose)
}

// SetCounts updates badges keyed by destination; zero/absent hides the badge.
func (s *Sidebar) SetCounts(counts map[string]int) {
	for _, section := range s.sections {
		section.SetCounts(counts)
	}
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
	s.SetActive(destination)
	for _, fn := range s.callbacks {
		if fn != nil {
			fn(destination)
		}
	}
}

func (s *Sidebar) AddContentWidget(widget *qt.QWidget) {
	if widget != nil {
		s.contentLayout.AddWidget(widget)
	}
}

// AddStretch inserts flexible vertical space between navigation groups.
func (s *Sidebar) AddStretch() {
	s.contentLayout.AddStretch()
}

func (s *Sidebar) AddSection(section *SidebarSection) {
	if section == nil {
		return
	}
	s.sections = append(s.sections, section)
	section.OnActivated(s.Activate)
	s.AddContentWidget(section.QWidget)
}

func (s *Sidebar) refreshStyle() {
	s.SetStyleSheet(fmt.Sprintf(
		"background-color: %s;",
		theme.CssColor(theme.ColorSurface),
	))
}
