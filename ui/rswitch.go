package ui

import (
	"time"

	"rapid/theme"

	qt "github.com/mappu/miqt/qt6"
)

const (
	switchWidth  = 36
	switchHeight = 20
	knobSize     = 16
	knobPad      = 2
)

type RSwitch struct {
	*qt.QCheckBox
	progress      float64 // 0..1, knob slide position
	timer         *qt.QTimer
	activeColor   *qt.QColor
	inactiveColor *qt.QColor
	animationFrom float64
	animationAt   time.Time
}

func NewRSwitch(checked bool) *RSwitch {
	s := &RSwitch{
		QCheckBox:     qt.NewQCheckBox2(),
		activeColor:   theme.ColorPrimary,
		inactiveColor: theme.ColorBorder,
		animationAt:   time.Now(),
	}
	s.SetFixedSize2(switchWidth, switchHeight)
	s.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	s.SetFocusPolicy(qt.StrongFocus)
	s.SetChecked(checked)
	if checked {
		s.progress = 1
	}

	// A small timer keeps the animation deterministic without depending on
	// QVariant marshaling through MIQT. Progress is elapsed-time based, so the
	// duration remains 120 ms even if a timer tick is delayed.
	s.timer = qt.NewQTimer2(s.QObject)
	s.timer.SetInterval(16)
	s.timer.OnTimeout(s.step)

	s.OnToggled(func(on bool) {
		s.animationFrom = s.progress
		s.animationAt = time.Now()
		if !s.timer.IsActive() {
			s.timer.Start2()
		}
	})

	// QCheckBox without text only reacts to clicks inside its small contents
	// rect; the whole pill is the switch, so toggle on any release.
	s.OnMouseReleaseEvent(func(super func(*qt.QMouseEvent), e *qt.QMouseEvent) {
		if s.IsEnabled() {
			s.SetChecked(!s.IsChecked())
		}
	})

	s.OnPaintEvent(func(super func(*qt.QPaintEvent), e *qt.QPaintEvent) {
		s.paint()
	})
	return s
}

func (s *RSwitch) SetActiveColor(color *qt.QColor) {
	if color == nil {
		color = theme.ColorPrimary
	}
	s.activeColor = color
	s.Update()
}

func (s *RSwitch) SetInactiveColor(color *qt.QColor) {
	if color == nil {
		color = theme.ColorBorder
	}
	s.inactiveColor = color
	s.Update()
}

func (s *RSwitch) ActiveColor() *qt.QColor {
	return s.activeColor
}

func (s *RSwitch) InactiveColor() *qt.QColor {
	return s.inactiveColor
}

func (s *RSwitch) step() {
	target := 0.0
	if s.IsChecked() {
		target = 1
	}
	const duration = 120 * time.Millisecond
	elapsed := time.Since(s.animationAt)
	fraction := float64(elapsed) / float64(duration)
	if fraction >= 1 {
		s.progress = target
		s.timer.Stop()
	} else if fraction < 0 {
		s.progress = s.animationFrom
	} else {
		s.progress = s.animationFrom + (target-s.animationFrom)*fraction
	}
	s.Update()
}

func (s *RSwitch) paint() {
	p := qt.NewQPainter2(s.QWidget.QPaintDevice)
	defer p.End()
	p.SetRenderHint(qt.QPainter__Antialiasing)

	background := blend(theme.ColorSurface, s.activeColor, s.progress)
	border := background.DarkerWithInt(120)
	if !s.IsEnabled() {
		background = blend(theme.ColorTextMuted, background, 0.5)
		border = background.DarkerWithInt(120)
	}
	p.SetPen(border)
	p.SetBrush(qt.NewQBrush3(background))
	p.DrawRoundedRect2(0, 0, switchWidth-1, switchHeight-1, switchHeight/2, switchHeight/2)

	knobX := knobPad + int(s.progress*float64(switchWidth-knobSize-2*knobPad))
	p.SetPenWithStyle(qt.NoPen)
	p.SetBrush(qt.NewQBrush3(theme.ColorText))
	p.DrawRoundedRect2(knobX, knobPad, knobSize, knobSize, knobSize/2, knobSize/2)
}

// blend lerps two colors by t (0..1).
func blend(a, b *qt.QColor, t float64) *qt.QColor {
	return qt.NewQColor3(lerp(a.Red(), b.Red(), t), lerp(a.Green(), b.Green(), t), lerp(a.Blue(), b.Blue(), t))
}

func lerp(a, b int, t float64) int {
	return int(float64(a) + (float64(b)-float64(a))*t)
}
