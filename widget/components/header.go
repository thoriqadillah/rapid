package components

import (
	"fmt"

	"rapid/widget/theme"
	"rapid/widget/ui"

	qt "github.com/mappu/miqt/qt6"
)

const defaultHeaderSearchWidth = 350

// Header is the top row containing menu, search, and New actions.
type Header struct {
	*qt.QWidget
	SearchField *ui.RTextField
	MenuButton  *ui.RButton
	AddButton   *ui.RButton

	preferredSearchWidth int
	addCallbacks         []func()
	menuCallbacks        []func()
}

func NewHeader() *Header {
	h := &Header{
		QWidget:              qt.NewQWidget3(nil, 0),
		preferredSearchWidth: defaultHeaderSearchWidth,
	}
	headerHeight := theme.TouchTarget + 2*theme.SpacingSm
	h.SetMinimumHeight(headerHeight)
	h.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
	h.SetStyleSheet(fmt.Sprintf(
		"background-color: %s; border-bottom: 1px solid %s;",
		theme.CssColor(theme.ColorBackground), theme.CssColor(theme.ColorBorder),
	))

	h.MenuButton = ui.NewRButtonIcon("MdiLightMenu.svg", ui.GhostVariant, false)
	h.MenuButton.OnClicked(func() {
		for _, fn := range h.menuCallbacks {
			if fn != nil {
				fn()
			}
		}
	})

	h.SearchField = ui.NewRTextField()
	h.SearchField.SetPlaceholder("Search...")
	h.SearchField.SetPrefixIcon(ui.IconPath("MdiLightMagnify.svg"))
	h.SearchField.SetMinimumWidth(0)
	h.SearchField.SetMaximumWidth(defaultHeaderSearchWidth)
	h.SearchField.SetMinimumHeight(theme.TouchTarget)

	h.AddButton = ui.NewRButton("New", ui.PrimaryVariant, false)
	h.AddButton.SetIconSource(ui.IconPath("MdiLightPlus.svg"))
	h.AddButton.SetMinimumHeight(theme.TouchTarget)
	h.AddButton.OnClicked(func() {
		for _, fn := range h.addCallbacks {
			if fn != nil {
				fn()
			}
		}
	})

	row := qt.NewQHBoxLayout2()
	row.SetContentsMargins(theme.SpacingSm, theme.SpacingSm, theme.SpacingSm, theme.SpacingSm)
	row.SetSpacing(theme.SpacingSm)
	row.AddWidget(h.MenuButton.QWidget)
	row.AddStretch()
	row.AddWidget(h.SearchField.QWidget)
	row.AddWidget(h.AddButton.QWidget)
	h.SetLayout(row.QLayout)
	return h
}

func (h *Header) OnAddClicked(fn func()) {
	if fn != nil {
		h.addCallbacks = append(h.addCallbacks, fn)
	}
}

func (h *Header) OnMenuClicked(fn func()) {
	if fn != nil {
		h.menuCallbacks = append(h.menuCallbacks, fn)
	}
}

func (h *Header) SetSearchText(v string) {
	h.SearchField.SetText(v)
}

func (h *Header) SearchText() string {
	return h.SearchField.Text()
}

func (h *Header) SetPreferredSearchWidth(v int) {
	if v < 0 {
		v = 0
	}
	h.preferredSearchWidth = v
	h.SearchField.SetMaximumWidth(v)
}

func (h *Header) PreferredSearchWidth() int {
	return h.preferredSearchWidth
}
