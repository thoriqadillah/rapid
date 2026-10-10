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
	searchField *ui.RTextField
	menuButton  *ui.RButton
	addButton   *ui.RButton

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

	h.menuButton = ui.NewRButtonIcon("MdiLightMenu.svg", ui.GhostVariant, false)
	h.menuButton.OnClicked(func() {
		for _, fn := range h.menuCallbacks {
			if fn != nil {
				fn()
			}
		}
	})

	h.searchField = ui.NewRTextField()
	h.searchField.SetPlaceholder("Search...")
	h.searchField.SetPrefixIcon(ui.IconPath("MdiLightMagnify.svg"))
	h.searchField.SetMinimumWidth(0)
	h.searchField.SetMaximumWidth(defaultHeaderSearchWidth)
	h.searchField.SetMinimumHeight(theme.TouchTarget)

	h.addButton = ui.NewRButton("New", ui.PrimaryVariant, false)
	h.addButton.SetIconSource(ui.IconPath("MdiLightPlus.svg"))
	h.addButton.SetMinimumHeight(theme.TouchTarget)
	h.addButton.OnClicked(func() {
		for _, fn := range h.addCallbacks {
			if fn != nil {
				fn()
			}
		}
	})

	row := qt.NewQHBoxLayout2()
	row.SetContentsMargins(theme.SpacingSm, theme.SpacingSm, theme.SpacingSm, theme.SpacingSm)
	row.SetSpacing(theme.SpacingSm)
	row.AddWidget(h.menuButton.QWidget)
	row.AddStretch()
	row.AddWidget(h.searchField.QWidget)
	row.AddWidget(h.addButton.QWidget)
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

func (h *Header) SetPreferredSearchWidth(v int) {
	v = min(v, 0)
	h.preferredSearchWidth = v
	h.searchField.SetMaximumWidth(v)
}

func (h *Header) PreferredSearchWidth() int {
	return h.preferredSearchWidth
}
