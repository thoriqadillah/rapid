package downloads

import (
	"rapid/lib/helpers/bools"
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

const progressBarHeight = 10

// ProgressBar is the painted pill track + fill used by rows and detail. It is
// painted directly (no stylesheet) so per-row category colors can't leak to
// siblings.
type ProgressBar struct {
	*qt.QWidget
	percent float64
	color   *qt.QColor
	error   bool
}

func NewProgressBar(parent *qt.QWidget) *ProgressBar {
	p := &ProgressBar{
		QWidget: qt.NewQWidget3(parent, 0),
		color:   theme.ColorCategoryUnknown,
	}
	p.SetMinimumHeight(progressBarHeight)
	p.SetMaximumHeight(progressBarHeight)
	p.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
	p.SetAttribute(qt.WA_TransparentForMouseEvents)
	p.OnPaintEvent(func(super func(*qt.QPaintEvent), event *qt.QPaintEvent) {
		super(event)
		p.paint()
	})
	return p
}

func (p *ProgressBar) SetValue(percent float64, color *qt.QColor, isError bool) {
	percent = max(percent, 0)
	percent = min(percent, 1)
	color = bools.Ternary(color != nil, color, theme.ColorCategoryUnknown)

	p.percent = percent
	p.color = color
	p.error = isError
	p.Update()
}

func (p *ProgressBar) paint() {
	w, h := p.Width(), p.Height()
	if w <= 0 || h <= 0 {
		return
	}
	painter := qt.NewQPainter2(p.QPaintDevice)
	defer painter.Delete()
	painter.SetRenderHint2(qt.QPainter__Antialiasing, true)
	radius := float64(h) / 2

	// A QPainter copies the pen/brush it is given, so each temporary can be
	// released right after use. MIQT has no finalizer on these constructors, so
	// skipping the Delete() leaks native memory on every repaint.
	borderPen := qt.NewQPen3(theme.ColorBorder)
	trackBrush := qt.NewQBrush3(theme.ColorSurface)
	painter.SetPenWithPen(borderPen)
	painter.SetBrush(trackBrush)
	painter.DrawRoundedRect2(0, 0, w-1, h-1, radius, radius)
	borderPen.Delete()
	trackBrush.Delete()

	fillW := int(float64(w-2) * p.percent)
	if fillW > 0 {
		fill := p.color
		if p.error {
			fill = theme.ColorDanger
		}
		fillBrush := qt.NewQBrush3(fill)
		painter.SetPenWithStyle(qt.NoPen)
		painter.SetBrush(fillBrush)
		painter.DrawRoundedRect2(1, 1, fillW, h-2, radius, radius)
		fillBrush.Delete()
	}
}
