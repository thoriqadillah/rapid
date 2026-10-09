package ui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/widget/theme"
)

func TestEmbeddedIconResource(t *testing.T) {
	for _, path := range []string{":/icons/MdiLightPlus.svg", "qrc:/icons/MdiLightPlus.svg", "qrc:/MdiLightPlus.svg"} {
		pixmap := TintedPixmap(path, theme.ColorPrimary, theme.IconMd)
		require.True(t, pixmap != nil && !pixmap.IsNull(), "embedded Qt resource %q did not rasterize", path)
	}
}

func TestTintedPixmapIsCached(t *testing.T) {
	path := IconPath("MdiLightPlus.svg")
	first := TintedPixmap(path, theme.ColorPrimary, theme.IconMd)
	second := TintedPixmap(path, theme.ColorPrimary, theme.IconMd)
	require.True(t, first != nil && first == second, "identical icon requests must share one cached pixmap")
	// A different color is a different key and must not alias the first pixmap.
	other := TintedPixmap(path, theme.ColorDanger, theme.IconMd)
	require.True(t, other != nil && other != first, "different colors must not share a pixmap")
}

func TestTintedPixmapAndLoadIconChecked(t *testing.T) {
	path := IconPath("MdiLightPlus.svg")
	require.NotEmpty(t, path, "bundled icon path is empty")
	require.Nil(t, TintedPixmap("", theme.ColorPrimary, theme.IconMd), "empty path should return nil")
	require.Nil(t, TintedPixmap(path, theme.ColorPrimary, 0), "non-positive size should return nil")
	require.Nil(t, TintedPixmap(path, nil, theme.IconMd), "nil color should return nil")

	pixmap := TintedPixmap(path, theme.ColorPrimary, theme.IconMd)
	require.True(t, pixmap != nil && !pixmap.IsNull(), "valid SVG did not rasterize")
	require.Equal(t, theme.IconMd, pixmap.Width(), "pixmap width")
	require.Equal(t, theme.IconMd, pixmap.Height(), "pixmap height")
	large := TintedPixmap(path, theme.ColorPrimary, theme.IconXl)
	require.True(t, large != nil && !large.IsNull() && large.Width() == theme.IconXl && large.Height() == theme.IconXl,
		"SVG was not decoded at the requested larger size")
	icon, err := LoadIconChecked(path, theme.ColorPrimary, theme.IconMd)
	require.NoError(t, err, "load valid icon")
	require.True(t, icon != nil && !icon.IsNull(), "valid SVG produced a null icon")
	_, err = LoadIconChecked(path, theme.ColorPrimary, 0)
	require.Error(t, err, "invalid size should fail")
	_, err = LoadIconChecked("missing.svg", theme.ColorPrimary, theme.IconMd)
	require.Error(t, err, "missing path should fail")
}
