package ui

import (
	"fmt"

	"rapid/theme"

	qt "github.com/mappu/miqt/qt6"
)

type RTextField struct {
	*qt.QWidget
	label         *qt.QLabel
	field         *qt.QLineEdit
	errorLabel    *qt.QLabel
	fieldRow      *qt.QHBoxLayout
	prefixPix     *qt.QPixmap
	suffixPix     *qt.QPixmap
	prefixSource  string
	suffixSource  string
	errorText     string
	iconSize      int
	iconColor     *qt.QColor
	selectByMouse bool
}

func NewRTextField() *RTextField {
	w := &RTextField{
		QWidget:       qt.NewQWidget3(nil, 0),
		label:         qt.NewQLabel3(""),
		field:         qt.NewQLineEdit(nil),
		errorLabel:    qt.NewQLabel3(""),
		fieldRow:      qt.NewQHBoxLayout2(),
		iconSize:      theme.IconMd,
		iconColor:     theme.ColorTextMuted,
		selectByMouse: true,
	}

	w.label.SetStyleSheet(fmt.Sprintf("color: %s; font-size: %dpx; margin: 0; padding: 0;", theme.CssColor(theme.ColorText), theme.TextSize))
	w.label.SetFixedHeight(theme.TextSize)
	w.label.SetAlignment(qt.AlignLeft | qt.AlignBottom)
	w.errorLabel.SetStyleSheet(fmt.Sprintf("color: %s; font-size: %dpx;", theme.CssColor(theme.ColorDanger), theme.TextSizeSm))
	w.errorLabel.SetAlignment(qt.AlignLeft | qt.AlignTop)
	w.errorLabel.SetWordWrap(true)

	w.field.SetMinimumHeight(theme.TouchTarget)
	w.field.SetFocusPolicy(qt.StrongFocus)
	w.SetFocusProxy(w.field.QWidget)
	w.SetFocusPolicy(qt.StrongFocus)
	w.field.OnPaintEvent(func(super func(*qt.QPaintEvent), e *qt.QPaintEvent) {
		super(e)
		w.drawSideIcons()
	})
	w.OnMousePressEvent(func(super func(*qt.QMouseEvent), e *qt.QMouseEvent) {
		super(e)
		w.field.SetFocus()
	})

	w.fieldRow.SetContentsMargins(0, 0, 0, 0)
	w.fieldRow.SetSpacing(theme.SpacingSm)
	w.fieldRow.AddWidget2(w.field.QWidget, 1)

	box := qt.NewQVBoxLayout2()
	box.SetContentsMargins(0, 0, 0, 0)
	box.SetSpacing(theme.SpacingSm)
	box.AddWidget(w.label.QWidget)
	box.AddLayout(w.fieldRow.QLayout)
	box.AddWidget(w.errorLabel.QWidget)
	box.AddStretch()

	w.QWidget.SetLayout(box.QLayout)
	w.label.SetVisible(false)
	w.errorLabel.SetVisible(false)
	w.applyStyle()
	return w
}

func (w *RTextField) applyStyle() {
	border := theme.ColorBorder
	if w.errorText != "" {
		border = theme.ColorDanger
	}
	left, right := theme.SpacingMd, theme.SpacingMd
	if w.prefixPix != nil {
		left += w.iconSize + theme.SpacingSm
	}
	if w.suffixPix != nil {
		right += w.iconSize + theme.SpacingSm
	}
	// Keep the danger border while focused; QSS otherwise lets QLineEdit:focus
	// overwrite the validation state.
	focusBorder := theme.ColorPrimary
	if w.errorText != "" {
		focusBorder = theme.ColorDanger
	}
	w.field.SetStyleSheet(fmt.Sprintf(`
		QLineEdit {
			background-color: %s;
			color: %s;
			border: 1px solid %s;
			border-radius: %dpx;
			padding: 0px %dpx 0px %dpx;
			min-height: %dpx;
			font-size: %dpx;
		}
		QLineEdit:focus {
			border-color: %s;
		}
	`, theme.CssColor(theme.ColorInputBackground), theme.CssColor(theme.ColorText), theme.CssColor(border), theme.RadiusSm, right, left, theme.TouchTarget, theme.TextSize, theme.CssColor(focusBorder)))
}

func (w *RTextField) SetText(text string) {
	w.field.SetText(text)
}

func (w *RTextField) Text() string {
	return w.field.Text()
}

func (w *RTextField) SetPlaceholder(text string) {
	w.field.SetPlaceholderText(text)
}

func (w *RTextField) SetLabel(text string) {
	w.label.SetText(text)
	w.label.SetVisible(text != "")
}

func (w *RTextField) SetError(text string) {
	w.errorText = text
	w.errorLabel.SetText(text)
	w.errorLabel.SetVisible(text != "")
	w.applyStyle()
}

func (w *RTextField) Error() string {
	return w.errorText
}

func (w *RTextField) SetPassword(on bool) {
	if on {
		w.SetEchoMode(qt.QLineEdit__Password)
	} else {
		w.SetEchoMode(qt.QLineEdit__Normal)
	}
}

func (w *RTextField) SetEchoMode(mode qt.QLineEdit__EchoMode) {
	w.field.SetEchoMode(mode)
}

func (w *RTextField) EchoMode() qt.QLineEdit__EchoMode {
	return w.field.EchoMode()
}

func (w *RTextField) SetReadOnly(readOnly bool) {
	w.field.SetReadOnly(readOnly)
}

func (w *RTextField) SetMaxLength(maxLength int) {
	w.field.SetMaxLength(maxLength)
}

func (w *RTextField) SetValidator(validator *qt.QValidator) {
	w.field.SetValidator(validator)
}

func (w *RTextField) SetInputMethodHints(hints qt.InputMethodHint) {
	w.field.SetInputMethodHints(hints)
}

// QLineEdit always supports mouse selection; retain the QML-facing state so
// callers can express the same contract without adding a custom text editor.
func (w *RTextField) SetSelectByMouse(on bool) {
	w.selectByMouse = on
}

func (w *RTextField) SelectByMouse() bool {
	return w.selectByMouse
}

func (w *RTextField) SetIconSize(size int) {
	if size <= 0 {
		size = theme.IconMd
	}
	w.iconSize = size
	w.reloadIcons()
	w.applyStyle()
}

func (w *RTextField) IconSize() int {
	return w.iconSize
}

func (w *RTextField) SetIconColor(color *qt.QColor) {
	if color == nil {
		color = theme.ColorTextMuted
	}
	w.iconColor = color
	w.reloadIcons()
}

func (w *RTextField) IconColor() *qt.QColor {
	return w.iconColor
}

func (w *RTextField) OnTextChanged(fn func(string)) {
	w.field.OnTextChanged(fn)
}

func (w *RTextField) FieldWidget() *qt.QWidget {
	return w.field.QWidget
}

func (w *RTextField) FieldLayout() *qt.QHBoxLayout {
	return w.fieldRow
}

func (w *RTextField) AddFieldWidget(widget *qt.QWidget) {
	if widget != nil {
		w.fieldRow.AddWidget(widget)
	}
}

// SetPrefixIcon / SetSuffixIcon place a tinted icon inside the field edges,
// like the QML prefixIcon/suffixIcon (default iconColor = colorTextMuted).
func (w *RTextField) SetPrefixIcon(path string) {
	w.prefixSource = path
	w.prefixPix = TintedPixmap(path, w.iconColor, w.iconSize)
	w.applyStyle()
	w.field.Update()
}

func (w *RTextField) SetSuffixIcon(path string) {
	w.suffixSource = path
	w.suffixPix = TintedPixmap(path, w.iconColor, w.iconSize)
	w.applyStyle()
	w.field.Update()
}

func (w *RTextField) reloadIcons() {
	w.prefixPix = TintedPixmap(w.prefixSource, w.iconColor, w.iconSize)
	w.suffixPix = TintedPixmap(w.suffixSource, w.iconColor, w.iconSize)
	w.field.Update()
}

// drawSideIcons paints prefix/suffix icons at their exact rendered size,
// since QLineEdit scales action icons to an internal default we can't set.
func (w *RTextField) drawSideIcons() {
	p := qt.NewQPainter2(w.field.QPaintDevice)
	defer p.End()
	fh := w.field.Height()
	if w.prefixPix != nil {
		p.DrawPixmap9(theme.SpacingMd, (fh-w.prefixPix.Height())/2, w.prefixPix)
	}
	if w.suffixPix != nil {
		p.DrawPixmap9(w.field.Width()-theme.SpacingMd-w.suffixPix.Width(), (fh-w.suffixPix.Height())/2, w.suffixPix)
	}
}
