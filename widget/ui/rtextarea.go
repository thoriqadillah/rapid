package ui

import (
	"fmt"

	"rapid/lib/reactive"
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

// RTextArea is the multiline twin of RTextField: a themed QTextEdit with the
// same label, placeholder, error and reactive value contract.
type RTextArea struct {
	*qt.QWidget
	label      *qt.QLabel
	area       *qt.QTextEdit
	errorLabel *qt.QLabel
	monospace  bool
	value      reactive.Model[string]
	error      reactive.Model[string]
	selfEdit   bool // true while syncing out an edit this widget originated
}

func NewRTextArea() *RTextArea {
	w := &RTextArea{
		QWidget:    qt.NewQWidget3(nil, 0),
		label:      qt.NewQLabel3(""),
		area:       qt.NewQTextEdit2(),
		errorLabel: qt.NewQLabel3(""),
		value:      reactive.NewSignal(""),
		error:      reactive.NewSignal(""),
	}

	w.label.SetStyleSheet(fmt.Sprintf("color: %s; font-size: %dpx; margin: 0; padding: 0; background: transparent;", theme.CssColor(theme.ColorText), theme.TextSize))
	w.label.SetFixedHeight(theme.TextSize)
	w.label.SetAlignment(qt.AlignLeft | qt.AlignBottom)
	w.errorLabel.SetStyleSheet(fmt.Sprintf("color: %s; font-size: %dpx; background: transparent;", theme.CssColor(theme.ColorDanger), theme.TextSizeSm))
	w.errorLabel.SetAlignment(qt.AlignLeft | qt.AlignTop)
	w.errorLabel.SetWordWrap(true)

	w.area.SetMinimumHeight(theme.TouchTarget * 3)
	w.area.SetFocusPolicy(qt.StrongFocus)
	w.area.SetAcceptRichText(false)
	w.area.OnTextChanged(func() {
		if s := w.area.ToPlainText(); w.value.Get() != s {
			// Everything downstream (Batch flush included) fires inside this
			// call, so the display effect can spot and skip the echo that
			// would otherwise rewrite the widget and yank the cursor.
			w.selfEdit = true
			w.value.Set(s)
			w.selfEdit = false
		}
	})

	box := qt.NewQVBoxLayout2()
	box.SetContentsMargins(0, 0, 0, 0)
	box.SetSpacing(theme.SpacingSm)
	box.AddWidget(w.label.QWidget)
	box.AddWidget2(w.area.QWidget, 1)
	box.AddWidget(w.errorLabel.QWidget)

	w.QWidget.SetLayout(box.QLayout)
	w.SetFocusPolicy(qt.StrongFocus)
	w.SetFocusProxy(w.area.QWidget)
	w.label.SetVisible(false)
	w.errorLabel.SetVisible(false)
	w.applyStyle()
	return w
}

func (w *RTextArea) applyStyle() {
	border := theme.ColorBorder
	if w.error.Peek() != "" {
		border = theme.ColorDanger
	}
	// Keep the danger border while focused; QSS otherwise lets
	// QTextEdit:focus overwrite the validation state.
	focusBorder := theme.ColorPrimary
	if w.error.Peek() != "" {
		focusBorder = theme.ColorDanger
	}
	fontFamily, fontSize := "", theme.TextSize
	if w.monospace {
		fontFamily, fontSize = "font-family: monospace; ", theme.TextSizeSm
	}
	w.area.SetStyleSheet(fmt.Sprintf(`
		QTextEdit {
			background-color: %s;
			color: %s;
			border: 1px solid %s;
			border-radius: %dpx;
			padding: %dpx;
			%sfont-size: %dpx;
		}
		QTextEdit:focus {
			border-color: %s;
		}
	`,
		theme.CssColor(theme.ColorInputBackground),
		theme.CssColor(theme.ColorText),
		theme.CssColor(border),
		theme.RadiusSm,
		theme.SpacingMd,
		fontFamily,
		fontSize,
		theme.CssColor(focusBorder),
	))
}

func (w *RTextArea) SetPlaceholder(text string) {
	w.area.SetPlaceholderText(text)
}

func (w *RTextArea) SetText(text string) {
	w.area.SetPlainText(text)
}

func (w *RTextArea) Text() string {
	return w.area.ToPlainText()
}

func (w *RTextArea) Bind(model reactive.Model[string]) {
	modelValue, cleanup := reactive.Bind(model)
	w.value = modelValue
	unsub := reactive.Effect(func() {
		text := w.value.Get() // must stay tracked, even on skipped runs
		if w.selfEdit || w.area.ToPlainText() == text {
			return
		}
		w.area.SetPlainText(text)
	})

	w.area.OnDestroyed(func() {
		unsub()
		cleanup()
	})
}

func (w *RTextArea) SetLabel(text string) {
	w.label.SetText(text)
	w.label.SetVisible(text != "")
}

func (w *RTextArea) SetError(text string) {
	w.error.Set(text)
	w.errorLabel.SetText(text)
	w.errorLabel.SetVisible(text != "")
	w.applyStyle()
}

func (w *RTextArea) BindError(text reactive.Model[string]) {
	errorModel, cleanup := reactive.Bind(text)
	w.error = errorModel
	unsub := reactive.Effect(func() {
		w.SetError(text.Get())
	})
	w.area.OnDestroyed(func() {
		unsub()
		cleanup()
	})
}

func (w *RTextArea) Error() string {
	return w.error.Peek()
}

// SetMonospace renders the text in the monospace/TextSizeSm style used by raw
// JSON and code panes.
func (w *RTextArea) SetMonospace(on bool) {
	w.monospace = on
	w.applyStyle()
}

func (w *RTextArea) SetReadOnly(readOnly bool) {
	w.area.SetReadOnly(readOnly)
}

func (w *RTextArea) OnTextChanged(fn func(string)) {
	w.area.OnTextChanged(func() {
		fn(w.area.ToPlainText())
	})
}

func (w *RTextArea) SetFocus() {
	w.area.SetFocus()
}

func (w *RTextArea) AreaWidget() *qt.QWidget {
	return w.area.QWidget
}
