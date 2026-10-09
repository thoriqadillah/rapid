package downloads

import (
	"rapid/lib"
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

const speedChartHeight = 120

// SpeedChart is the DownloadSpeedSample equivalent: a surface panel with an
// area-filled sparkline, or a flat baseline when there are no samples.
type SpeedChart struct {
	*qt.QWidget
	samples  []int64
	category lib.Category
}

func NewSpeedChart(parent *qt.QWidget) *SpeedChart {
	c := &SpeedChart{
		QWidget:  qt.NewQWidget3(parent, 0),
		category: lib.CategoryUnknown,
	}
	c.SetMinimumHeight(speedChartHeight)
	c.SetMaximumHeight(speedChartHeight)
	c.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Fixed)
	c.SetAttribute(qt.WA_TransparentForMouseEvents)
	c.OnPaintEvent(func(super func(*qt.QPaintEvent), event *qt.QPaintEvent) {
		super(event)
		c.paint()
	})
	return c
}

func (c *SpeedChart) SetSamples(samples []int64) {
	c.samples = append([]int64(nil), samples...)
	c.Update()
}

func (c *SpeedChart) Samples() []int64 {
	return append([]int64(nil), c.samples...)
}

func (c *SpeedChart) SetCategory(category lib.Category) {
	c.category = category
	c.Update()
}

func (c *SpeedChart) Category() lib.Category {
	return c.category
}

func (c *SpeedChart) paint() {
	w, h := c.Width(), c.Height()
	if w <= 0 || h <= 0 {
		return
	}
	painter := qt.NewQPainter2(c.QPaintDevice)
	defer painter.Delete()
	painter.SetRenderHint2(qt.QPainter__Antialiasing, true)

	// Surface panel background. Every NewQPen3/NewQBrush3/NewQPainterPath is an
	// unfinalized native allocation in MIQT, so each one is released explicitly.
	panelBrush := qt.NewQBrush3(theme.ColorSurface)
	painter.SetPenWithStyle(qt.NoPen)
	painter.SetBrush(panelBrush)
	painter.DrawRoundedRect2(0, 0, w, h, float64(theme.RadiusSm), float64(theme.RadiusSm))
	panelBrush.Delete()

	dpr := c.DevicePixelRatioF()
	if dpr <= 0 {
		dpr = 1
	}
	lineWidth := 2 * dpr

	if len(c.samples) == 0 {
		pen := qt.NewQPen3(theme.ColorTextMuted)
		pen.SetWidthF(lineWidth)
		painter.SetPenWithPen(pen)
		painter.DrawLine2(1, h-1, w-1, h-1)
		pen.Delete()
		return
	}

	const pad = 1
	plotW := float64(w - 2*pad)
	plotH := float64(h - 2*pad)
	maxSpeed := int64(1)
	for _, sample := range c.samples {
		if sample > maxSpeed {
			maxSpeed = sample
		}
	}
	stepX := 0.0
	if len(c.samples) > 1 {
		stepX = plotW / float64(len(c.samples)-1)
	}
	xAt := func(i int) float64 { return float64(pad) + float64(i)*stepX }
	yAt := func(v int64) float64 {
		return float64(pad) + plotH - plotH*float64(v)/float64(maxSpeed)
	}

	categoryColor := theme.CategoryColor(c.category)

	fill := qt.NewQPainterPath()
	defer fill.Delete()
	fill.MoveTo2(float64(pad), float64(h-pad))
	for i, sample := range c.samples {
		fill.LineTo2(xAt(i), yAt(sample))
	}
	fill.LineTo2(xAt(len(c.samples)-1), float64(h-pad))
	fill.CloseSubpath()
	fillBrush := qt.NewQBrush3(theme.WithAlpha(categoryColor, 38))
	painter.SetPenWithStyle(qt.NoPen)
	painter.FillPath(fill, fillBrush)
	fillBrush.Delete()

	stroke := qt.NewQPainterPath()
	defer stroke.Delete()
	for i, sample := range c.samples {
		if i == 0 {
			stroke.MoveTo2(xAt(i), yAt(sample))
		} else {
			stroke.LineTo2(xAt(i), yAt(sample))
		}
	}
	pen := qt.NewQPen3(categoryColor)
	defer pen.Delete()
	pen.SetWidthF(lineWidth)
	pen.SetJoinStyle(qt.RoundJoin)
	// DrawPath fills with the current brush as well as stroking; clear it so the
	// stroke doesn't paint the surface color over the area fill.
	painter.SetBrushWithStyle(qt.NoBrush)
	painter.SetPenWithPen(pen)
	painter.DrawPath(stroke)
}
