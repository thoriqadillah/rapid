package ui

import (
	"fmt"

	"rapid/lib/reactive"
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
	progress float64 // 0..1, knob slide position
	anim     *qt.QVariantAnimation
	value    reactive.Model[bool]
}

func NewRSwitch(checked bool) *RSwitch {
	s := &RSwitch{
		QCheckBox: qt.NewQCheckBox2(),
		value:     reactive.NewSignal(false),
	}
	s.SetFixedSize2(switchWidth, switchHeight)
	s.SetStyleSheet(fmt.Sprintf(`
		QCheckBox::indicator {
			width: %dpx;
			height: %dpx;
		}`,
		switchWidth,
		switchHeight,
	))
	cursor := qt.NewQCursor2(qt.PointingHandCursor)
	s.SetCursor(cursor)
	cursor.Delete()
	s.SetFocusPolicy(qt.StrongFocus)
	s.SetChecked(checked)
	if checked {
		s.progress = 1
	}

	s.anim = qt.NewQVariantAnimation2(s.QObject)
	s.anim.SetDuration(knobAnimDuration)
	easingCurve := qt.NewQEasingCurve3(qt.QEasingCurve__InOutCubic)
	s.anim.SetEasingCurve(easingCurve) // the animation copies the curve
	easingCurve.Delete()
	s.anim.OnValueChanged(func(v *qt.QVariant) {
		s.progress = v.ToDouble()
		s.Update()
	})

	s.OnToggled(s.toggle)
	s.OnPaintEvent(func(super func(*qt.QPaintEvent), e *qt.QPaintEvent) {
		s.paint()
	})
	return s
}

func (s *RSwitch) Bind(model reactive.Model[bool]) {
	modelValue, cleanup := reactive.Bind(model)
	s.value = modelValue
	s.OnToggled(s.value.Set)
	unsub := reactive.Effect(func() {
		s.SetChecked(s.value.Get())
	})

	s.OnDestroyed(func() {
		cleanup()
		unsub()
	})
}

func (s *RSwitch) paint() {
	p := qt.NewQPainter2(s.QWidget.QPaintDevice)
	defer p.Delete()
	p.SetRenderHint(qt.QPainter__Antialiasing)

	background := blend(theme.ColorSurface, theme.ColorPrimary, s.progress)
	if !s.IsEnabled() {
		disabled := blend(theme.ColorTextMuted, background, 0.5)
		background.Delete()
		background = disabled
	}
	border := background.DarkerWithInt(120)

	trackBrush := qt.NewQBrush3(background)
	p.SetPen(border)
	p.SetBrush(trackBrush)
	p.DrawRoundedRect2(0, 0, switchWidth-1, switchHeight-1, switchHeight/2, switchHeight/2)
	trackBrush.Delete()
	background.Delete()

	knobX := knobPad + int(s.progress*float64(switchWidth-knobSize-2*knobPad))
	knobBrush := qt.NewQBrush3(theme.ColorText)
	p.SetPenWithStyle(qt.NoPen)
	p.SetBrush(knobBrush)
	p.DrawRoundedRect2(knobX, knobPad, knobSize, knobSize, knobSize/2, knobSize/2)
	knobBrush.Delete()
}

func (s *RSwitch) toggle(on bool) {
	to := 0.0
	if on {
		to = 1
	}
	// SetStartValue/SetEndValue copy their QVariant, so free both at once.
	start := qt.NewQVariant9(s.progress)
	end := qt.NewQVariant9(to)
	s.anim.SetStartValue(start)
	s.anim.SetEndValue(end)
	start.Delete()
	end.Delete()
	s.anim.Start()
}

// blend lerps two colors by t (0..1).
func blend(a, b *qt.QColor, t float64) *qt.QColor {
	return qt.NewQColor3(lerp(a.Red(), b.Red(), t), lerp(a.Green(), b.Green(), t), lerp(a.Blue(), b.Blue(), t))
}

func lerp(a, b int, t float64) int {
	return int(float64(a) + (float64(b)-float64(a))*t)
}
