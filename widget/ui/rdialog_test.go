package ui

import (
	"testing"

	qt "github.com/mappu/miqt/qt6"
)

func TestRDialogStructureAndSizing(t *testing.T) {
	owner := qt.NewQWidget2()
	dialog := NewRDialog(owner)
	if dialog.MinimumWidth() != 500 {
		t.Fatalf("minimum width = %d, want 500", dialog.MinimumWidth())
	}
	if dialog.BodyLayout == nil || dialog.FooterLayout == nil {
		t.Fatal("dialog layouts are nil")
	}
	if dialog.FooterLayout.Count() != 1 {
		t.Fatalf("footer should start with one alignment stretch, got %d items", dialog.FooterLayout.Count())
	}
	dialog.AddBodyWidget(qt.NewQLabel3("body").QWidget)
	dialog.AddFooterWidget(qt.NewQPushButton3("OK").QWidget)
	if dialog.BodyLayout.Count() != 1 || dialog.FooterLayout.Count() != 2 {
		t.Fatal("dialog helper methods did not add widgets")
	}
	dialog.SetMaxHeight(320)
	if dialog.MaxHeight() != 320 || dialog.MaximumHeight() != 320 {
		t.Fatal("maximum height was not applied")
	}
	dialog.SetMaxHeight(0)
	if dialog.MaxHeight() != 0 {
		t.Fatal("zero maximum height should clear the custom limit")
	}
}

func TestRDialogOpenForAndOverlay(t *testing.T) {
	owner := qt.NewQWidget2()
	owner.SetGeometry(0, 0, 640, 480)
	owner.Show()
	defer owner.Hide()

	dialog := NewRDialog(owner)
	opened := 0
	dialog.OnOpened(func() { opened++ })
	dialog.OpenFor(owner)
	qt.QCoreApplication_ProcessEvents()
	if opened != 1 {
		t.Fatalf("opened callbacks = %d, want 1", opened)
	}
	if !dialog.IsVisible() {
		t.Fatal("OpenFor did not show dialog")
	}
	if dialog.OverlayWidget() == nil || !dialog.OverlayWidget().IsVisible() {
		t.Fatal("OpenFor did not show owner overlay")
	}
	if dialog.OverlayWidget().Width() != owner.Width() || dialog.OverlayWidget().Height() != owner.Height() {
		t.Fatal("overlay does not cover owner geometry")
	}
	dialog.Hide()
	qt.QCoreApplication_ProcessEvents()
	if dialog.OverlayWidget().IsVisible() {
		t.Fatal("hiding dialog did not clean up overlay")
	}
}
