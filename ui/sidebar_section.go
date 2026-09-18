package ui

import (
	"rapid/theme"

	qt "github.com/mappu/miqt/qt6"
)

// SidebarItemData is the typed replacement for the QML object-array model.
type SidebarItemData struct {
	Destination  string
	Label        string
	Count        string
	IconSource   string
	IconColor    *qt.QColor
	CategoryItem bool
}

// SidebarSection owns its item widgets and forwards item destinations.
type SidebarSection struct {
	*qt.QWidget
	ItemsLayout *qt.QVBoxLayout

	headingLabel *SidebarLabel
	heading      string
	items        []SidebarItemData
	itemWidgets  []*SidebarItem
	current      string
	topMargin    int
	callbacks    []func(string)
}

func NewSidebarSection() *SidebarSection {
	s := &SidebarSection{
		QWidget:      qt.NewQWidget3(nil, 0),
		ItemsLayout:  qt.NewQVBoxLayout2(),
		headingLabel: NewSidebarLabel(""),
	}
	s.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Preferred)
	s.ItemsLayout.SetContentsMargins(0, 0, 0, 0)
	s.ItemsLayout.SetSpacing(theme.SpacingXs)
	s.ItemsLayout.AddWidget(s.headingLabel.QWidget)
	s.headingLabel.SetVisible(false)
	s.SetLayout(s.ItemsLayout.QLayout)
	return s
}

func (s *SidebarSection) SetHeading(v string) {
	s.heading = v
	s.headingLabel.SetText(v)
	s.headingLabel.SetVisible(v != "")
}

func (s *SidebarSection) Heading() string {
	return s.heading
}

func (s *SidebarSection) HeadingVisible() bool {
	return s.heading != ""
}

func (s *SidebarSection) SetItems(items []SidebarItemData) {
	s.items = append([]SidebarItemData(nil), items...)
	for s.ItemsLayout.Count() > 1 {
		item := s.ItemsLayout.TakeAt(s.ItemsLayout.Count() - 1)
		if item != nil && item.Widget() != nil {
			item.Widget().Hide()
		}
	}
	s.itemWidgets = s.itemWidgets[:0]
	for _, data := range s.items {
		item := NewSidebarItem(data.Destination, data.Label)
		item.SetCount(data.Count)
		item.SetIconSource(data.IconSource)
		item.SetIconColor(data.IconColor)
		item.SetCategoryItem(data.CategoryItem)
		item.SetSelected(data.Destination == s.current)
		destination := data.Destination
		item.OnActivated(func() { s.emitActivated(destination) })
		s.ItemsLayout.AddWidget(item.QWidget)
		s.itemWidgets = append(s.itemWidgets, item)
	}
}

func (s *SidebarSection) Items() []SidebarItemData {
	return append([]SidebarItemData(nil), s.items...)
}

func (s *SidebarSection) SetCurrentDestination(v string) {
	s.current = v
	for _, item := range s.itemWidgets {
		item.SetSelected(item.Destination() == v)
	}
}

func (s *SidebarSection) CurrentDestination() string {
	return s.current
}

func (s *SidebarSection) SetTopMargin(v int) {
	if v < 0 {
		v = 0
	}
	s.topMargin = v
	s.ItemsLayout.SetContentsMargins(0, v, 0, 0)
}

func (s *SidebarSection) TopMargin() int {
	return s.topMargin
}

func (s *SidebarSection) OnActivated(fn func(string)) {
	if fn != nil {
		s.callbacks = append(s.callbacks, fn)
	}
}

func (s *SidebarSection) emitActivated(destination string) {
	for _, fn := range s.callbacks {
		if fn != nil {
			fn(destination)
		}
	}
}

func (s *SidebarSection) Refresh() {
	s.SetCurrentDestination(s.current)
}

func (s *SidebarSection) ItemWidgets() []*SidebarItem {
	return append([]*SidebarItem(nil), s.itemWidgets...)
}
