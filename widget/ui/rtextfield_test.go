package ui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/widget/theme"

	qt "github.com/mappu/miqt/qt6"
)

func TestRTextFieldBasicStateAndLayout(t *testing.T) {
	field := NewRTextField()
	field.SetLabel("URL")
	require.True(t, !field.label.IsHidden() && field.label.Text() == "URL", "label was not shown")
	field.SetText("https://example.com")
	require.Equal(t, "https://example.com", field.Text(), "text")
	field.SetPlaceholder("Paste URL")
	require.Equal(t, "Paste URL", field.field.PlaceholderText(), "placeholder was not applied")
	field.SetError("Invalid URL")
	require.True(t, field.Error() == "Invalid URL" && !field.errorLabel.IsHidden(), "error state was not shown")
	require.Contains(t, field.field.StyleSheet(), theme.CssColor(theme.ColorDanger), "error stylesheet does not contain danger color")
	field.SetError("")
	require.True(t, field.errorLabel.IsHidden(), "empty error should hide error label")
	require.True(t, field.FieldWidget() == field.field.QWidget && field.FieldLayout() != nil,
		"field accessors are not wired to the underlying widgets")
	proxy := field.FocusProxy()
	require.True(t, proxy != nil && proxy.UnsafePointer() == field.field.QWidget.UnsafePointer(),
		"wrapper focus proxy is not the line edit")
}

func TestRTextFieldIconsPreserveErrorAndSupportProperties(t *testing.T) {
	field := NewRTextField()
	path := IconPath("MdiLightContentPaste.svg")
	field.SetError("Invalid URL")
	field.SetPrefixIcon(path)
	require.True(t, field.prefixPix != nil && field.Error() == "Invalid URL",
		"prefix icon was not loaded or error state was lost")
	require.Contains(t, field.field.StyleSheet(), theme.CssColor(theme.ColorDanger), "adding an icon cleared the error border")
	field.SetSuffixIcon(path)
	require.NotNil(t, field.suffixPix, "suffix icon was not loaded")
	field.SetIconSize(theme.IconLg)
	require.True(t, field.prefixPix.Width() == theme.IconLg && field.suffixPix.Width() == theme.IconLg,
		"icon size was not applied to both sides")
	customColor := qt.NewQColor6("#abcdef")
	field.SetIconColor(customColor)
	require.Equal(t, customColor.Name(), field.IconColor().Name(), "custom icon color was not retained")
	field.SetPrefixIcon("")
	field.SetSuffixIcon("")
	require.True(t, field.prefixPix == nil && field.suffixPix == nil, "empty icon sources should clear icons")
	require.Contains(t, field.field.StyleSheet(), theme.CssColor(theme.ColorDanger), "clearing icons cleared the error border")
}

func TestRTextFieldInputContract(t *testing.T) {
	field := NewRTextField()
	field.SetPassword(true)
	require.Equal(t, qt.QLineEdit__Password, field.EchoMode(), "password mode was not applied")
	field.SetPassword(false)
	require.Equal(t, qt.QLineEdit__Normal, field.EchoMode(), "normal echo mode was not restored")
	field.SetReadOnly(true)
	require.True(t, field.field.IsReadOnly(), "read-only state was not applied")
	field.SetMaxLength(12)
	require.Equal(t, 12, field.field.MaxLength(), "maximum length was not applied")
	field.SetSelectByMouse(false)
	require.False(t, field.SelectByMouse(), "select-by-mouse state was not retained")
	changed := ""
	field.OnTextChanged(func(text string) { changed = text })
	field.SetText("abc")
	require.Equal(t, "abc", changed, "text changed callback value")
}
