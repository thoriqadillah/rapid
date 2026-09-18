package ui

import (
	"testing"
	"time"

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
	if switchWidget.timer == nil || switchWidget.timer.Interval() != 16 {
		t.Fatal("switch animation timer was not configured")
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
	if switchWidget.animationAt.IsZero() {
		t.Fatal("checked transition did not record animation start")
	}
	switchWidget.animationAt = time.Now().Add(-200 * time.Millisecond)
	switchWidget.step()
	if switchWidget.progress != 1 || switchWidget.timer.IsActive() {
		t.Fatal("switch animation did not reach checked state")
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
