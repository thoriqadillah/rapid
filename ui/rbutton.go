package ui

import (
	"fmt"

	"rapid/theme"

	qt "github.com/mappu/miqt/qt6"
)

type RButtonVariant int

const (
	BaseVariant RButtonVariant = iota
	PrimaryVariant
	InfoVariant
	SecondaryVariant
	GhostVariant
	WarningVariant
	DangerVariant
	LinkVariant
)

type RButton struct {
	*qt.QPushButton
	variant      RButtonVariant
	outlined     bool
	link         bool
	iconOnly     bool
	iconSize     int
	cornerRadius int
	text         string
	iconSource   string
	hovered      bool
}

func NewRButton(text string, variant RButtonVariant, outlined bool) *RButton {
	b := &RButton{
		QPushButton:  qt.NewQPushButton3(text),
		variant:      variant,
		outlined:     outlined,
		link:         variant == LinkVariant,
		iconSize:     theme.IconMd,
		cornerRadius: theme.RadiusSm,
		text:         text,
	}
	b.SetMinimumHeight(theme.TouchTarget)
	b.setStyleSheet()
	b.SetCursor(qt.NewQCursor2(qt.PointingHandCursor))
	b.SetMouseTracking(true)
	b.OnEnterEvent(func(super func(*qt.QEnterEvent), event *qt.QEnterEvent) {
		super(event)
		b.hovered = true
		b.refreshIcon()
		b.setStyleSheet()
	})
	b.OnLeaveEvent(func(super func(*qt.QEvent), event *qt.QEvent) {
		super(event)
		b.hovered = false
		b.refreshIcon()
		b.setStyleSheet()
	})
	return b
}

func NewRButtonIcon(iconSource string, variant RButtonVariant, outlined bool) *RButton {
	b := NewRButton("", variant, outlined)
	b.SetIconSource(IconPath(iconSource))
	b.SetIconOnly(true)
	return b
}

// SetText keeps the original text so icon-only mode can be toggled safely.
func (b *RButton) SetText(text string) {
	b.text = text
	if !b.iconOnly {
		b.QPushButton.SetText(text)
	}
}

func (b *RButton) Text() string {
	return b.text
}

// SetIconSource sets a monochrome SVG icon tinted to the button's current
// foreground color. Empty paths remove the icon.
func (b *RButton) SetIconSource(path string) {
	b.iconSource = path
	b.refreshIcon()
}

func (b *RButton) SetIconSize(size int) {
	if size <= 0 {
		size = theme.IconMd
	}
	b.iconSize = size
	b.refreshIcon()
	b.setStyleSheet()
}

func (b *RButton) SetIconOnly(on bool) {
	b.iconOnly = on
	if on {
		b.QPushButton.SetText("")
	} else {
		b.QPushButton.SetText(b.text)
	}
	b.setStyleSheet()
}

func (b *RButton) SetLink(link bool) {
	b.link = link
	b.setStyleSheet()
}

func (b *RButton) SetTooltip(text string) {
	b.SetToolTip(text)
	if text == "" {
		b.SetToolTipDuration(-1)
		return
	}
	b.SetToolTipDuration(500)
}

func (b *RButton) SetCornerRadius(radius int) {
	if radius < 0 {
		radius = 0
	}
	b.cornerRadius = radius
	b.setStyleSheet()
}

func (b *RButton) SetEnabled(enabled bool) {
	b.QPushButton.SetEnabled(enabled)
	b.refreshIcon()
}

func (b *RButton) foregroundColor() *qt.QColor {
	switch b.variant {
	case PrimaryVariant, DangerVariant:
		return theme.ColorTextInverted
	default:
		return theme.ColorText
	}
}

func (b *RButton) iconColor() *qt.QColor {
	if b.hovered && b.IsEnabled() {
		return theme.ColorTextInverted
	}
	return b.foregroundColor()
}

func (b *RButton) refreshIcon() {
	b.SetIcon(LoadIcon(b.iconSource, b.iconColor(), b.iconSize))
	b.QPushButton.SetIconSize(qt.NewQSize2(b.iconSize, b.iconSize))
}

func (b *RButton) setStyleSheet() {
	bg := theme.ColorButtonBase
	hoverBg := theme.ColorButtonBaseHover
	variantColor := theme.ColorButtonBase
	fg := b.foregroundColor()
	hoverFg := fg
	border := "1px solid transparent"
	paddingH := theme.SpacingXl
	if b.iconOnly {
		paddingH = theme.SpacingSm
	}

	switch b.variant {
	case PrimaryVariant:
		bg, hoverBg, variantColor = theme.ColorPrimary, theme.ColorPrimaryHover, theme.ColorPrimary
	case InfoVariant:
		bg, hoverBg, variantColor = theme.ColorInfo, theme.ColorInfoHover, theme.ColorInfo
	case SecondaryVariant:
		bg, hoverBg, variantColor = theme.ColorAccent, theme.ColorAccentHover, theme.ColorAccent
	case GhostVariant:
		bg, hoverBg, variantColor = nil, theme.ColorButtonBaseHover, theme.ColorButtonBase
	case WarningVariant:
		bg, hoverBg, variantColor = theme.ColorWarning, theme.ColorWarningHover, theme.ColorWarning
	case DangerVariant:
		bg, hoverBg, variantColor = theme.ColorDanger, theme.ColorDangerHover, theme.ColorDanger
	case LinkVariant:
		// LinkVariant starts in link mode, but SetLink can explicitly override it.
	}

	if b.link {
		b.SetStyleSheet(fmt.Sprintf(`
			QPushButton {
				background-color: transparent;
				color: %s;
				border: none;
				min-height: %dpx;
				font-size: %dpx;
				outline: none;
			}
			QPushButton:focus {
				text-decoration: underline;
			}
			QPushButton:hover {
				color: %s;
				text-decoration: underline;
			}
			QPushButton:disabled {
				color: %s;
				background-color: transparent;
				opacity: 0.5;
			}
		`, theme.CssColor(fg),
			theme.TouchTarget,
			theme.TextSize,
			theme.CssColor(theme.ColorTextInverted),
			theme.CssColor(theme.ColorTextMuted)),
		)
		return
	}

	if b.outlined {
		border = "1px solid " + theme.CssColor(variantColor)
		bg = nil
		hoverBg = variantColor
		hoverFg = theme.ColorTextInverted
	}
	if b.hovered && b.IsEnabled() {
		hoverFg = theme.ColorTextInverted
	}

	b.SetStyleSheet(fmt.Sprintf(`
		QPushButton {
			background-color: %s;
			color: %s;
			border: %s;
			border-radius: %dpx;
			padding: 0 %dpx;
			min-height: %dpx;
			font-size: %dpx;
			outline: none;
		}
		QPushButton:focus {
			border: %s;
		}
		QPushButton:hover {
			background-color: %s;
			color: %s;
		}
		QPushButton:disabled {
			background-color: %s;
			color: %s;
			opacity: 0.5;
		}
	`, theme.CssColor(bg),
		theme.CssColor(fg),
		border,
		b.cornerRadius,
		paddingH,
		theme.TouchTarget,
		theme.TextSize,
		border,
		theme.CssColor(hoverBg),
		theme.CssColor(hoverFg),
		theme.CssColor(variantColor),
		theme.CssColor(theme.ColorTextMuted),
	),
	)
}
