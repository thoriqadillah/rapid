package theme

import (
	"fmt"

	qt "github.com/mappu/miqt/qt6"
)

// Structural colors are filled in by Init from the running application's
// system palette; Init must run after qt.NewQApplication. Until then they are
// nil (and CssColor renders "transparent").
var (
	ColorBackground *qt.QColor
	ColorSurface    *qt.QColor
	ColorBorder     *qt.QColor

	ColorPrimary      *qt.QColor
	ColorPrimaryHover *qt.QColor

	ColorButtonBase      *qt.QColor
	ColorButtonBaseHover *qt.QColor

	ColorText         *qt.QColor
	ColorTextInverted *qt.QColor
	ColorTextMuted    *qt.QColor

	ColorInputBackground *qt.QColor
)

// Semantic status and file-category colors are fixed: they signal meaning
// (error, warning, audio, video...) and have no OS palette equivalent.
var (
	ColorDanger      = color("#CB3A2A")
	ColorDangerHover = ColorDanger.DarkerWithInt(110)

	ColorAccent      = color("#FF79C6")
	ColorAccentHover = ColorAccent.DarkerWithInt(110)

	ColorSuccess      = color("#50FA7B")
	ColorSuccessHover = ColorSuccess.DarkerWithInt(110)

	ColorWarning      = color("#FFB86C")
	ColorWarningHover = ColorWarning.DarkerWithInt(110)

	ColorInfo      = color("#8BE9FD")
	ColorInfoHover = ColorInfo.DarkerWithInt(110)

	ColorCategoryAudio       = color("#50FA7B")
	ColorCategoryApplication = color("#F1FA8C")
	ColorCategoryCompressed  = color("#FF5555")
	ColorCategoryDocument    = color("#8BE9FD")
	ColorCategoryImage       = color("#BD93F9")
	ColorCategoryVideo       = color("#FFB86C")
	ColorCategoryUnknown     = color("#F8F8F2")
)

func Init() {
	p := qt.QGuiApplication_Palette()

	ColorBackground = p.ColorWithCr(qt.QPalette__Window)
	ColorSurface = p.ColorWithCr(qt.QPalette__Base)
	ColorBorder = p.ColorWithCr(qt.QPalette__Mid)

	ColorPrimary = p.ColorWithCr(qt.QPalette__Highlight)
	ColorPrimaryHover = ColorPrimary.DarkerWithInt(110)

	ColorText = p.ColorWithCr(qt.QPalette__WindowText)
	ColorTextInverted = p.ColorWithCr(qt.QPalette__HighlightedText)
	ColorTextMuted = withAlpha(ColorText, 0x99) // 60% alpha

	ColorButtonBase = p.ColorWithCr(qt.QPalette__Base)
	ColorButtonBaseHover = ColorButtonBase.DarkerWithInt(110)

	ColorInputBackground = ColorSurface
}

func withAlpha(c *qt.QColor, alpha int) *qt.QColor {
	out := qt.NewQColor10(c)
	out.SetAlpha(alpha)
	return out
}

const (
	TextSize   = 14
	TextSizeSm = 10
)

const (
	RadiusSm    = 4
	RadiusMd    = 8
	RadiusLg    = 12
	RadiusXl    = 14
	RadiusPill  = 20
	RadiusSheet = 22
)

const (
	SpacingXs = 4
	SpacingSm = 8
	SpacingMd = 12
	SpacingLg = 16
	SpacingXl = 24
)

const (
	BreakpointMd = 600
	BreakpointLg = 840
	TouchTarget  = 32
)

const (
	IconXs = 10
	IconSm = 18
	IconMd = 24
	IconLg = 32
	IconXl = 48
)

func color(h string) *qt.QColor {
	return qt.NewQColor6(h)
}

// CssColor returns a color as a CSS string; nil means CSS "transparent",
// so themed components can render "no color" without a separate sentinel.
func CssColor(c *qt.QColor) string {
	if c == nil {
		return "transparent"
	}
	switch c.Spec() {
	case qt.QColor__Rgb:
		return fmt.Sprintf("rgba(%d, %d, %d, %d)", c.Red(), c.Green(), c.Blue(), c.Alpha())
	}

	return c.Name()
}

func CategoryColor(category string) *qt.QColor {
	switch category {
	case "audio":
		return ColorCategoryAudio
	case "application":
		return ColorCategoryApplication
	case "compressed":
		return ColorCategoryCompressed
	case "document":
		return ColorCategoryDocument
	case "image":
		return ColorCategoryImage
	case "video":
		return ColorCategoryVideo
	default:
		return ColorCategoryUnknown
	}
}
