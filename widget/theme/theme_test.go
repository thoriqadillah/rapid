package theme

import (
	"os"
	"testing"

	qt "github.com/mappu/miqt/qt6"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if os.Getenv("QT_QPA_PLATFORM") == "" {
		_ = os.Setenv("QT_QPA_PLATFORM", "offscreen")
	}
	qt.NewQApplication([]string{"rapid-theme-tests"})
	Init()
	os.Exit(m.Run())
}

func TestInitMapsSystemPalette(t *testing.T) {
	for name, c := range map[string]*qt.QColor{
		"background": ColorBackground,
		"surface":    ColorSurface,
		"border":     ColorBorder,
		"primary":    ColorPrimary,
		"accent":     ColorAccent,
		"text":       ColorText,
		"inverted":   ColorTextInverted,
		"button":     ColorButtonBase,
		"input":      ColorInputBackground,
	} {
		require.NotNil(t, c, "%s color not initialized from palette", name)
	}
	require.Equal(t, 0x99, ColorTextMuted.Alpha(), "muted alpha")
}
