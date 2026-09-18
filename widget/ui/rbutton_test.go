package ui

import (
	"strings"
	"testing"

	"rapid/widget/theme"
)

func TestRButtonVariantsAndStyling(t *testing.T) {
	variants := []RButtonVariant{
		BaseVariant, PrimaryVariant, InfoVariant, SecondaryVariant,
		GhostVariant, WarningVariant, DangerVariant, LinkVariant,
	}
	for _, variant := range variants {
		button := NewRButton("Action", variant, false)
		if button.MinimumHeight() < theme.TouchTarget {
			t.Errorf("variant %d minimum height %d < touch target", variant, button.MinimumHeight())
		}
		if button.StyleSheet() == "" {
			t.Errorf("variant %d has no stylesheet", variant)
		}
	}

	base := NewRButton("Base", BaseVariant, false)
	if !strings.Contains(base.StyleSheet(), "padding: 0 24px") {
		t.Fatal("text button does not have QML-equivalent horizontal padding")
	}

	ghost := NewRButton("Ghost", GhostVariant, false)
	if !strings.Contains(ghost.StyleSheet(), "background-color: transparent") {
		t.Fatal("ghost button is not transparent")
	}

	link := NewRButton("Link", BaseVariant, false)
	link.SetLink(true)

	outlined := NewRButton("Outline", PrimaryVariant, true)
	if !strings.Contains(outlined.StyleSheet(), "1px solid") {
		t.Fatal("outlined button has no border")
	}
	outlined.SetCornerRadius(13)
	if !strings.Contains(outlined.StyleSheet(), "border-radius: 13px") {
		t.Fatal("custom corner radius was not applied")
	}
}

func TestRButtonIconAndInteractionContract(t *testing.T) {
	button := NewRButton("New", PrimaryVariant, false)
	path := IconPath("MdiLightPlus.svg")
	button.SetIconSource(path)
	button.SetIconSize(theme.IconLg)
	if got := button.QPushButton.IconSize().Width(); got != theme.IconLg {
		t.Fatalf("icon width = %d, want %d", got, theme.IconLg)
	}
	if button.Icon().IsNull() {
		t.Fatal("valid button icon is null")
	}
	button.SetIconOnly(true)
	if button.QPushButton.Text() != "" || button.Text() != "New" {
		t.Fatal("icon-only mode did not hide visible text while preserving it")
	}
	if button.MinimumHeight() < theme.TouchTarget {
		t.Fatal("icon-only button lost touch-target height")
	}
	button.SetIconOnly(false)
	if button.QPushButton.Text() != "New" {
		t.Fatal("turning off icon-only mode did not restore text")
	}
	button.SetTooltip("Create item")
	if button.ToolTip() != "Create item" || button.ToolTipDuration() != 500 {
		t.Fatal("tooltip contract was not applied")
	}

	clicked := false
	button.OnClicked(func() { clicked = true })
	button.QPushButton.Click()
	if !clicked {
		t.Fatal("clicked callback did not run")
	}
	button.SetEnabled(false)
	if button.IsEnabled() {
		t.Fatal("button should be disabled")
	}
	button.SetTooltip("")
	if button.ToolTipDuration() != -1 {
		t.Fatal("empty tooltip should disable the tooltip delay")
	}
}
