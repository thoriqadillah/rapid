package components

import (
	"strconv"

	"rapid/lib/helpers/bools"
	"rapid/widget/theme"

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
	OnActivated  func()
}

// SidebarSection owns its item widgets and forwards item destinations.
type SidebarSection struct {
	*qt.QWidget
	itemsLayout *qt.QVBoxLayout

	headingLabel *SidebarLabel
	items        []SidebarItemData
	itemWidgets  []*SidebarItem
	topMargin    int
	activated    []func(string)
}

func NewSidebarSection() *SidebarSection {
	s := &SidebarSection{
		QWidget:      qt.NewQWidget3(nil, 0),
		itemsLayout:  qt.NewQVBoxLayout2(),
		headingLabel: NewSidebarLabel(""),
	}
	s.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Preferred)
	s.itemsLayout.SetContentsMargins(0, 0, 0, 0)
	s.itemsLayout.SetSpacing(theme.SpacingXs)
	s.itemsLayout.AddWidget(s.headingLabel.QWidget)
	s.headingLabel.SetVisible(false)
	s.SetLayout(s.itemsLayout.QLayout)
	return s
}

func (s *SidebarSection) SetHeading(v string) {
	s.headingLabel.SetText(v)
	s.headingLabel.SetVisible(v != "")
}

func (s *SidebarSection) SetItems(items []SidebarItemData) {
	s.items = append([]SidebarItemData(nil), items...)
	for s.itemsLayout.Count() > 1 {
		item := s.itemsLayout.TakeAt(s.itemsLayout.Count() - 1)
		if item == nil {
			continue
		}
		if widget := item.Widget(); widget != nil {
			widget.Hide()
			widget.DeleteLater()
		}
		item.Delete() // takeAt transfers ownership; the item leaks otherwise
	}
	s.itemWidgets = s.itemWidgets[:0]
	for _, data := range s.items {
		item := NewSidebarItem(data.Destination, data.Label)
		item.SetCount(data.Count)
		item.SetIconSource(data.IconSource)
		item.SetIconColor(data.IconColor)
		item.SetCategoryItem(data.CategoryItem)
		item.OnActivated(data.OnActivated)
		item.OnActivated(func() { s.emitActivated(data.Destination) })
		s.itemsLayout.AddWidget(item.QWidget)
		s.itemWidgets = append(s.itemWidgets, item)
	}
}

// SetCounts updates badges keyed by destination; zero/absent hides the badge.
func (s *SidebarSection) SetCounts(counts map[string]int) {
	for _, item := range s.itemWidgets {
		n := counts[item.destination]
		item.SetCount(bools.Ternary(n > 0, strconv.Itoa(n), ""))
	}
}

func (s *SidebarSection) SetActive(dest string) {
	for _, item := range s.itemWidgets {
		item.SetSelected(item.destination == dest)
	}
}

func (s *SidebarSection) SetTopMargin(v int) {
	v = min(v, 0)
	s.itemsLayout.SetContentsMargins(0, v, 0, 0)
}

// OnActivated registers a destination callback. Unlike per-item wiring, it
// survives SetItems because the section re-wires every item it creates.
func (s *SidebarSection) OnActivated(fn func(dest string)) {
	if fn != nil {
		s.activated = append(s.activated, fn)
	}
}

func (s *SidebarSection) emitActivated(dest string) {
	for _, fn := range s.activated {
		if fn != nil {
			fn(dest)
		}
	}
}
