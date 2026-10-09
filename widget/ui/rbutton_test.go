package ui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/widget/theme"
)

func TestRButtonVariantsAndStyling(t *testing.T) {
	variants := []RButtonVariant{
		BaseVariant, PrimaryVariant, InfoVariant, SecondaryVariant,
		GhostVariant, WarningVariant, DangerVariant, LinkVariant,
	}
	for _, variant := range variants {
		button := NewRButton("Action", variant, false)
		require.GreaterOrEqual(t, button.MinimumHeight(), theme.TouchTarget, "variant %d minimum height below touch target", variant)
		require.NotEmpty(t, button.StyleSheet(), "variant %d has no stylesheet", variant)
	}

	base := NewRButton("Base", BaseVariant, false)
	require.Contains(t, base.StyleSheet(), "padding: 0 24px", "text button does not have QML-equivalent horizontal padding")

	ghost := NewRButton("Ghost", GhostVariant, false)
	require.Contains(t, ghost.StyleSheet(), "background-color: transparent", "ghost button is not transparent")

	link := NewRButton("Link", BaseVariant, false)
	link.SetLink(true)

	outlined := NewRButton("Outline", PrimaryVariant, true)
	require.Contains(t, outlined.StyleSheet(), "1px solid", "outlined button has no border")
	outlined.SetCornerRadius(13)
	require.Contains(t, outlined.StyleSheet(), "border-radius: 13px", "custom corner radius was not applied")
}

func TestRButtonIconAndInteractionContract(t *testing.T) {
	button := NewRButton("New", PrimaryVariant, false)
	path := IconPath("MdiLightPlus.svg")
	button.SetIconSource(path)
	button.SetIconSize(theme.IconLg)
	require.Equal(t, theme.IconLg, button.QPushButton.IconSize().Width(), "icon width")
	require.False(t, button.Icon().IsNull(), "valid button icon is null")
	button.SetIconOnly(true)
	require.True(t, button.QPushButton.Text() == "" && button.Text() == "New",
		"icon-only mode did not hide visible text while preserving it")
	require.GreaterOrEqual(t, button.MinimumHeight(), theme.TouchTarget, "icon-only button lost touch-target height")
	button.SetIconOnly(false)
	require.Equal(t, "New", button.QPushButton.Text(), "turning off icon-only mode did not restore text")
	button.SetTooltip("Create item")
	require.True(t, button.ToolTip() == "Create item" && button.ToolTipDuration() == 500, "tooltip contract was not applied")

	clicked := false
	button.OnClicked(func() { clicked = true })
	button.QPushButton.Click()
	require.True(t, clicked, "clicked callback did not run")
	button.SetEnabled(false)
	require.False(t, button.IsEnabled(), "button should be disabled")
	button.SetTooltip("")
	require.Equal(t, -1, button.ToolTipDuration(), "empty tooltip should disable the tooltip delay")
}
