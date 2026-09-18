package ui

import (
	"testing"

	"rapid/widget/theme"
)

func TestEmbeddedIconResource(t *testing.T) {
	for _, path := range []string{":/icons/MdiLightPlus.svg", "qrc:/icons/MdiLightPlus.svg", "qrc:/MdiLightPlus.svg"} {
		pixmap := TintedPixmap(path, theme.ColorPrimary, theme.IconMd)
		if pixmap == nil || pixmap.IsNull() {
			t.Fatalf("embedded Qt resource %q did not rasterize", path)
		}
	}
}

func TestTintedPixmapAndLoadIconChecked(t *testing.T) {
	path := IconPath("MdiLightPlus.svg")
	if path == "" {
		t.Fatal("bundled icon path is empty")
	}
	if TintedPixmap("", theme.ColorPrimary, theme.IconMd) != nil {
		t.Fatal("empty path should return nil")
	}
	if TintedPixmap(path, theme.ColorPrimary, 0) != nil {
		t.Fatal("non-positive size should return nil")
	}
	if TintedPixmap(path, nil, theme.IconMd) != nil {
		t.Fatal("nil color should return nil")
	}

	pixmap := TintedPixmap(path, theme.ColorPrimary, theme.IconMd)
	if pixmap == nil || pixmap.IsNull() {
		t.Fatal("valid SVG did not rasterize")
	}
	if pixmap.Width() != theme.IconMd || pixmap.Height() != theme.IconMd {
		t.Fatalf("unexpected pixmap size: %dx%d", pixmap.Width(), pixmap.Height())
	}
	large := TintedPixmap(path, theme.ColorPrimary, theme.IconXl)
	if large == nil || large.IsNull() || large.Width() != theme.IconXl || large.Height() != theme.IconXl {
		t.Fatal("SVG was not decoded at the requested larger size")
	}
	icon, err := LoadIconChecked(path, theme.ColorPrimary, theme.IconMd)
	if err != nil {
		t.Fatalf("load valid icon: %v", err)
	}
	if icon == nil || icon.IsNull() {
		t.Fatal("valid SVG produced a null icon")
	}
	if _, err := LoadIconChecked(path, theme.ColorPrimary, 0); err == nil {
		t.Fatal("invalid size should fail")
	}
	if _, err := LoadIconChecked("missing.svg", theme.ColorPrimary, theme.IconMd); err == nil {
		t.Fatal("missing path should fail")
	}
}
