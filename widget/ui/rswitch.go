package ui

import (
	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

const (
	switchWidth      = 36
	switchHeight     = 20
	knobSize         = 16
	knobPad          = 2
	knobAnimDuration = 120
)

type RSwitch struct {
	*qt.QCheckBox
	progress      float64 // 0..1, knob slide position
	anim          *qt.QVariantAnimation
	activeColor   *qt.QColor
	inactiveColor *qt.QColor
}

func NewRSwitch(checked bool) *RSwitch {
	s := &RSwitch{
		QCheckBox:     qt.NewQCheckBox2(),
		activeColor:   theme.ColorPrimary,
		inactiveColor: theme.ColorBorder,
	}
	s.SetFixedSize2(switchWidth, switchHeight)
	s.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	s.SetFocusPolicy(qt.StrongFocus)
	s.SetChecked(checked)
	if checked {
		s.progress = 1
	}

	s.anim = qt.NewQVariantAnimation2(s.QObject)
	s.anim.SetDuration(knobAnimDuration)
	s.anim.SetEasingCurve(qt.NewQEasingCurve3(qt.QEasingCurve__InOutCubic))
	s.anim.OnValueChanged(func(v *qt.QVariant) {
		s.progress = v.ToDouble()
		s.Update()
	})

	s.OnToggled(func(on bool) {
		s.anim.SetStartValue(qt.NewQVariant9(s.progress))
		to := 0.0
		if on {
			to = 1
		}
		s.anim.SetEndValue(qt.NewQVariant9(to))
		s.anim.Start()
	})

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
