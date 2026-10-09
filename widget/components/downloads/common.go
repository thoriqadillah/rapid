package downloads

import (
	"fmt"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

// Animation durations, mirroring the QML transitions (detailHeight 150ms,
// detail opacity/slide 250ms, list add/remove 150ms).
const (
	addRemoveDuration    = 200
	detailHeightDuration = 200
	detailFadeDuration   = 300
)

// styledLabel builds a transparent QLabel with the given color and the theme
// text size.
func styledLabel(text string, color *qt.QColor) *qt.QLabel {
	l := qt.NewQLabel3(text)
	l.SetWordWrap(false)
	l.SetAttribute(qt.WA_TransparentForMouseEvents)
	l.SetStyleSheet(fmt.Sprintf(
		`QLabel {
			background: transparent;
			border: none;
			color: %s;
			font-size: %dpx;
		}`,
		theme.CssColor(color), theme.TextSize))
	return l
}

// smallLabel builds a transparent QLabel at the small text size.
func smallLabel(text string, color *qt.QColor) *qt.QLabel {
	l := styledLabel(text, color)
	l.SetStyleSheet(fmt.Sprintf(
		`QLabel {
			background: transparent;
			border: none;
			color: %s;
			font-size: %dpx;
		}`,
		theme.CssColor(color), theme.TextSizeSm))
	return l
}

// fixedLabel builds a styled label pinned to width, used to keep header and row
// columns aligned.
func fixedLabel(text string, color *qt.QColor, width int) *qt.QLabel {
	l := styledLabel(text, color)
	l.SetFixedWidth(width)
	l.SetAlignment(qt.AlignLeft | qt.AlignVCenter)
	return l
}

// elidedLabel is a QLabel that re-elides its full text on resize. QLabel has no
// elide mode in Qt Widgets, so we measure with QFontMetrics instead.
type elidedLabel struct {
	*qt.QLabel
	full     string
	mode     qt.TextElideMode
	lastW    int
	lastText string
	fm       *qt.QFontMetrics // owned; freed on destroy
}

func newElidedLabel(text string, color *qt.QColor, mode qt.TextElideMode) *elidedLabel {
	l := &elidedLabel{QLabel: qt.NewQLabel3(""), mode: mode, lastW: -1}
	l.SetWordWrap(false)
	l.SetAttribute(qt.WA_TransparentForMouseEvents)
	l.SetMinimumWidth(0)
	l.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Preferred)
	l.SetStyleSheet(fmt.Sprintf(`
		QLabel {
			background: transparent;
			border: none;
			color: %s;
		}`,
		theme.CssColor(color),
	))
	font := l.Font()
	font.SetPixelSize(theme.TextSize)
	l.SetFont(font)
	// Measure with a single QFontMetrics for the label's lifetime: creating one
	// per apply() leaked a native QFontMetrics on every resize and text update.
	l.fm = qt.NewQFontMetrics(font)
	l.OnDestroyed(func() {
		if l.fm != nil {
			l.fm.Delete()
			l.fm = nil
		}
	})
	l.OnResizeEvent(func(super func(*qt.QResizeEvent), e *qt.QResizeEvent) {
		super(e)
		// height-only resizes (detail expand animation) must not
		// re-elide: same width re-settles the layout into a feedback loop.
		if e != nil && e.Size() != nil && e.OldSize() != nil &&
			e.Size().Width() == e.OldSize().Width() {
			return
		}
		l.apply()
	})
	l.SetFullText(text)
	return l
}

func (l *elidedLabel) SetFullText(text string) {
	l.full = text
	l.apply()
}

func (l *elidedLabel) FullText() string {
	return l.full
}

func (l *elidedLabel) apply() {
	if l.fm == nil {
		return
	}
	width := max(l.Width(), 0)
	text := l.fm.ElidedText(l.full, l.mode, width)
	// same width + same output must not re-SetText: during the
	// expand animation every height tick would otherwise invalidate the
	// layout and vibrate the row.
	if width == l.lastW && text == l.lastText {
		return
	}
	l.lastW = width
	l.lastText = text
	l.SetText(text)
}

// roundedPanel paints a rounded background and/or border in its own paint
// event. Plain-QWidget stylesheet borders are unreliable on this MIQT backend,
// so panels that must show a border draw it themselves.
type roundedPanel struct {
	*qt.QWidget
	background  *qt.QColor
	border      *qt.QColor
	borderWidth int
	radius      int
}

func newRoundedPanel(parent *qt.QWidget, background, border *qt.QColor, borderWidth, radius int) *roundedPanel {
	p := &roundedPanel{
		QWidget:     qt.NewQWidget3(parent, 0),
		background:  background,
		border:      border,
		borderWidth: borderWidth,
		radius:      radius,
	}
	p.OnPaintEvent(func(super func(*qt.QPaintEvent), event *qt.QPaintEvent) {
		super(event)
		p.paint()
	})
	return p
}

func (p *roundedPanel) SetBorderColor(color *qt.QColor) {
	p.border = color
	p.Update()
}

func (p *roundedPanel) SetBackgroundColor(color *qt.QColor) {
	p.background = color
	p.Update()
}

func (p *roundedPanel) paint() {
	w, h := p.Width(), p.Height()
	if w <= 0 || h <= 0 {
		return
	}
	bw := 0
	if p.border != nil {
		bw = p.borderWidth
	}
	// Inset the path by half the pen width so an odd-width border lands on
	// whole pixels instead of being antialiased into a faint half-line.
	half := float64(bw) / 2
	w2 := float64(w) - float64(bw)
	h2 := float64(h) - float64(bw)
	if w2 <= 0 || h2 <= 0 {
		return
	}

	painter := qt.NewQPainter2(p.QPaintDevice)
	defer painter.Delete()
	painter.SetRenderHint2(qt.QPainter__Antialiasing, true)

	if p.background != nil {
		brush := qt.NewQBrush3(p.background)
		painter.SetBrush(brush)
		defer brush.Delete()
	} else {
		painter.SetBrushWithStyle(qt.NoBrush)
	}
	if p.border != nil && bw > 0 {
		pen := qt.NewQPen3(p.border)
		pen.SetWidth(bw)
		painter.SetPenWithPen(pen)
		defer pen.Delete()
	} else {
		painter.SetPenWithStyle(qt.NoPen)
	}
	path := qt.NewQPainterPath()
	defer path.Delete()
	path.AddRoundedRect2(half, half, w2, h2, float64(p.radius), float64(p.radius))
	painter.DrawPath(path)
}

// animateMaxHeight animates a widget's maximumHeight, which is layout-safe
// (unlike animating pos on a layout-managed child). Same QPropertyAnimation
// mechanism as Sidebar.animate. The animation is returned so callers can
// Stop a running one before starting its reverse (overlapping height anims
// on the same widget fight and vibrate the row).
func animateMaxHeight(w *qt.QWidget, from, to, duration int, easing qt.QEasingCurve__Type, done func()) *qt.QPropertyAnimation {
	if w == nil {
		return nil
	}
	w.SetMinimumHeight(0)
	w.SetMaximumHeight(from)
	anim := qt.NewQPropertyAnimation2(qt.UnsafeNewQObject(w.UnsafePointer()), []byte("maximumHeight"))
	anim.SetParent(w.QObject) // Qt owns the animation; no Go finalizer needed
	anim.SetDuration(duration)
	// QPropertyAnimation copies the value/easing it is given, so the QVariants
	// and QEasingCurve can be released immediately after Set*.
	start := qt.NewQVariant4(from)
	end := qt.NewQVariant4(to)
	easingCurve := qt.NewQEasingCurve3(easing)
	anim.SetStartValue(start)
	anim.SetEndValue(end)
	anim.SetEasingCurve(easingCurve)
	start.Delete()
	end.Delete()
	easingCurve.Delete()
	if done != nil {
		anim.OnFinished(done)
	}
	anim.Start()
	return anim
}

// animateOpacity animates a graphics opacity effect from -> to.
func animateOpacity(effect *qt.QGraphicsOpacityEffect, from, to float64, duration int, easing qt.QEasingCurve__Type, done func()) *qt.QPropertyAnimation {
	if effect == nil {
		return nil
	}
	anim := qt.NewQPropertyAnimation2(qt.UnsafeNewQObject(effect.UnsafePointer()), []byte("opacity"))
	anim.SetParent(effect.QObject) // Qt owns the animation; no Go finalizer needed
	anim.SetDuration(duration)
	start := qt.NewQVariant9(from)
	end := qt.NewQVariant9(to)
	easingCurve := qt.NewQEasingCurve3(easing)
	anim.SetStartValue(start)
	anim.SetEndValue(end)
	anim.SetEasingCurve(easingCurve)
	start.Delete()
	end.Delete()
	easingCurve.Delete()
	if done != nil {
		anim.OnFinished(done)
	}
	anim.Start()
	return anim
}
