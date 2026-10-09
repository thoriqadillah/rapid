package ui

import (
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"github.com/stretchr/testify/require"
)

func TestRDialogStructureAndSizing(t *testing.T) {
	owner := qt.NewQWidget2()
	dialog := NewRDialog(owner)
	require.Equal(t, 500, dialog.MinimumWidth(), "minimum width")
	require.True(t, dialog.BodyLayout != nil && dialog.FooterLayout != nil, "dialog layouts are nil")
	require.Equal(t, 1, dialog.FooterLayout.Count(), "footer should start with one alignment stretch")
	dialog.AddBodyWidget(qt.NewQLabel3("body").QWidget)
	dialog.AddFooterWidget(qt.NewQPushButton3("OK").QWidget)
	require.True(t, dialog.BodyLayout.Count() == 1 && dialog.FooterLayout.Count() == 2,
		"dialog helper methods did not add widgets")
	dialog.SetMaxHeight(320)
	require.True(t, dialog.MaxHeight() == 320 && dialog.MaximumHeight() == 320, "maximum height was not applied")
	dialog.SetMaxHeight(0)
	require.Equal(t, 0, dialog.MaxHeight(), "zero maximum height should clear the custom limit")
}

func TestRDialogOpenForAndOverlay(t *testing.T) {
	owner := qt.NewQWidget2()
	owner.SetGeometry(0, 0, 640, 480)
	owner.Show()
	defer owner.Hide()

	dialog := NewRDialog(owner)
	opened := 0
	dialog.OnOpened(func() { opened++ })
	dialog.Open()
	qt.QCoreApplication_ProcessEvents()
	require.Equal(t, 1, opened, "opened callbacks")
	require.True(t, dialog.IsVisible(), "Open did not show dialog")
	require.True(t, dialog.OverlayWidget() != nil && dialog.OverlayWidget().IsVisible(), "Open did not show owner overlay")
	require.True(t, dialog.OverlayWidget().Width() == owner.Width() && dialog.OverlayWidget().Height() == owner.Height(),
		"overlay does not cover owner geometry")
	dialog.Hide()
	qt.QCoreApplication_ProcessEvents()
	overlay := dialog.OverlayWidget()
	require.False(t, overlay != nil && overlay.IsVisible(), "hiding dialog did not clean up overlay")
}
