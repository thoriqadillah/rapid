package ui

import (
	"strings"
	"testing"

	"rapid/theme"

	qt "github.com/mappu/miqt/qt6"
)

func TestRTextFieldBasicStateAndLayout(t *testing.T) {
	field := NewRTextField()
	field.SetLabel("URL")
	if field.label.IsHidden() || field.label.Text() != "URL" {
		t.Fatal("label was not shown")
	}
	field.SetText("https://example.com")
	if field.Text() != "https://example.com" {
		t.Fatalf("text = %q", field.Text())
	}
	field.SetPlaceholder("Paste URL")
	if field.field.PlaceholderText() != "Paste URL" {
		t.Fatal("placeholder was not applied")
	}
	field.SetError("Invalid URL")
	if field.Error() != "Invalid URL" || field.errorLabel.IsHidden() {
		t.Fatal("error state was not shown")
	}
	if !strings.Contains(field.field.StyleSheet(), theme.CssColor(theme.ColorDanger)) {
		t.Fatal("error stylesheet does not contain danger color")
	}
	field.SetError("")
	if !field.errorLabel.IsHidden() {
		t.Fatal("empty error should hide error label")
	}
	if field.FieldWidget() != field.field.QWidget || field.FieldLayout() == nil {
		t.Fatal("field accessors are not wired to the underlying widgets")
	}
	proxy := field.FocusProxy()
	if proxy == nil || proxy.UnsafePointer() != field.field.QWidget.UnsafePointer() {
		t.Fatal("wrapper focus proxy is not the line edit")
	}
}

func TestRTextFieldIconsPreserveErrorAndSupportProperties(t *testing.T) {
	field := NewRTextField()
	path := IconPath("MdiLightContentPaste.svg")
	field.SetError("Invalid URL")
	field.SetPrefixIcon(path)
	if field.prefixPix == nil || field.Error() != "Invalid URL" {
		t.Fatal("prefix icon was not loaded or error state was lost")
	}
	if !strings.Contains(field.field.StyleSheet(), theme.CssColor(theme.ColorDanger)) {
		t.Fatal("adding an icon cleared the error border")
	}
	field.SetSuffixIcon(path)
	if field.suffixPix == nil {
		t.Fatal("suffix icon was not loaded")
	}
	field.SetIconSize(theme.IconLg)
	if field.prefixPix.Width() != theme.IconLg || field.suffixPix.Width() != theme.IconLg {
		t.Fatal("icon size was not applied to both sides")
	}
	customColor := qt.NewQColor6("#abcdef")
	field.SetIconColor(customColor)
	if field.IconColor().Name() != customColor.Name() {
		t.Fatal("custom icon color was not retained")
	}
	field.SetPrefixIcon("")
	field.SetSuffixIcon("")
	if field.prefixPix != nil || field.suffixPix != nil {
		t.Fatal("empty icon sources should clear icons")
	}
	if !strings.Contains(field.field.StyleSheet(), theme.CssColor(theme.ColorDanger)) {
		t.Fatal("clearing icons cleared the error border")
	}
}

func TestRTextFieldInputContract(t *testing.T) {
	field := NewRTextField()
	field.SetPassword(true)
	if field.EchoMode() != qt.QLineEdit__Password {
		t.Fatal("password mode was not applied")
	}
	field.SetPassword(false)
	if field.EchoMode() != qt.QLineEdit__Normal {
		t.Fatal("normal echo mode was not restored")
	}
	field.SetReadOnly(true)
	if !field.field.IsReadOnly() {
		t.Fatal("read-only state was not applied")
	}
	field.SetMaxLength(12)
	if field.field.MaxLength() != 12 {
		t.Fatal("maximum length was not applied")
	}
	field.SetSelectByMouse(false)
	if field.SelectByMouse() {
		t.Fatal("select-by-mouse state was not retained")
	}
	changed := ""
	field.OnTextChanged(func(text string) { changed = text })
	field.SetText("abc")
	if changed != "abc" {
		t.Fatalf("text changed callback value = %q", changed)
	}
}
