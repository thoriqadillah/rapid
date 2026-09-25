package ui

import (
	"testing"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

func TestRSwitchStateAndColors(t *testing.T) {
	switchWidget := NewRSwitch(true)
	if !switchWidget.IsChecked() {
		t.Fatal("switch did not preserve initial checked state")
	}
	if switchWidget.Width() != switchWidth || switchWidget.Height() != switchHeight {
		t.Fatalf("switch size = %dx%d, want %dx%d", switchWidget.Width(), switchWidget.Height(), switchWidth, switchHeight)
	}
	active := qt.NewQColor6("#112233")
	inactive := qt.NewQColor6("#445566")
	switchWidget.SetActiveColor(active)
	switchWidget.SetInactiveColor(inactive)
	if switchWidget.ActiveColor().Name() != active.Name() || switchWidget.InactiveColor().Name() != inactive.Name() {
		t.Fatal("custom switch colors were not retained")
	}
	if switchWidget.anim == nil || switchWidget.anim.Duration() != knobAnimDuration {
		t.Fatal("switch animation was not configured")
	}
}

func TestRSwitchAnimationAndToggle(t *testing.T) {
	switchWidget := NewRSwitch(false)
	var toggled bool
	switchWidget.OnToggled(func(on bool) { toggled = on })
	switchWidget.SetChecked(true)
	if !toggled {
		t.Fatal("checked transition did not emit toggled")
	}
	if got := switchWidget.anim.EndValue().ToDouble(); got != 1 {
		t.Fatalf("animation end value = %v, want 1", got)
	}
	if got := switchWidget.anim.StartValue().ToDouble(); got != 0 {
		t.Fatalf("animation start value = %v, want current progress 0", got)
	}
	if switchWidget.anim.State() != qt.QAbstractAnimation__Running {
		t.Fatal("switch animation did not start")
	}
	switchWidget.SetEnabled(false)
	if switchWidget.IsEnabled() {
		t.Fatal("switch should be disabled")
	}
	// The underlying Qt property remains programmatically controllable;
	// mouse/keyboard interaction is what the disabled state gates.
	if theme.CssColor(switchWidget.ActiveColor()) == "" {
		t.Fatal("active color became invalid")
	}
}
