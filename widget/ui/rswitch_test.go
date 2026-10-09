package ui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

func TestRSwitchStateAndColors(t *testing.T) {
	switchWidget := NewRSwitch(true)
	require.True(t, switchWidget.IsChecked(), "switch did not preserve initial checked state")
	require.Equal(t, switchWidth, switchWidget.Width(), "switch width")
	require.Equal(t, switchHeight, switchWidget.Height(), "switch height")
	active := qt.NewQColor6("#112233")
	inactive := qt.NewQColor6("#445566")
	switchWidget.SetActiveColor(active)
	switchWidget.SetInactiveColor(inactive)
	require.Equal(t, active.Name(), switchWidget.ActiveColor().Name(), "active color not retained")
	require.Equal(t, inactive.Name(), switchWidget.InactiveColor().Name(), "inactive color not retained")
	require.True(t, switchWidget.anim != nil && switchWidget.anim.Duration() == knobAnimDuration,
		"switch animation was not configured")
}

func TestRSwitchAnimationAndToggle(t *testing.T) {
	switchWidget := NewRSwitch(false)
	var toggled bool
	switchWidget.OnToggled(func(on bool) { toggled = on })
	switchWidget.SetChecked(true)
	require.True(t, toggled, "checked transition did not emit toggled")
	require.Equal(t, float64(1), switchWidget.anim.EndValue().ToDouble(), "animation end value")
	require.Equal(t, float64(0), switchWidget.anim.StartValue().ToDouble(), "animation start value")
	require.Equal(t, qt.QAbstractAnimation__Running, switchWidget.anim.State(), "switch animation did not start")
	switchWidget.SetEnabled(false)
	require.False(t, switchWidget.IsEnabled(), "switch should be disabled")
	// The underlying Qt property remains programmatically controllable;
	// mouse/keyboard interaction is what the disabled state gates.
	require.NotEmpty(t, theme.CssColor(switchWidget.ActiveColor()), "active color became invalid")
}
