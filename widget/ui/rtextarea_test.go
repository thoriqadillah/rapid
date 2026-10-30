package ui

import (
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/lib/reactive"
	"rapid/widget/theme"
)

func TestRTextAreaBasicStateAndLayout(t *testing.T) {
	area := NewRTextArea()
	area.SetLabel("JSON")
	require.True(t, !area.label.IsHidden() && area.label.Text() == "JSON", "label was not shown")
	area.SetText("{\n  \"a\": 1\n}")
	require.Equal(t, "{\n  \"a\": 1\n}", area.Text(), "text")
	require.Equal(t, "{\n  \"a\": 1\n}", area.value.Get(), "text edits should publish to the value model")
	area.SetPlaceholder("{}")
	require.Equal(t, "{}", area.area.PlaceholderText(), "placeholder was not applied")
	area.SetError("Invalid JSON")
	require.True(t, area.Error() == "Invalid JSON" && !area.errorLabel.IsHidden(), "error state was not shown")
	require.Contains(t, area.area.StyleSheet(), theme.CssColor(theme.ColorDanger), "error stylesheet does not contain danger color")
	area.SetError("")
	require.True(t, area.errorLabel.IsHidden(), "empty error should hide error label")
	require.True(t, area.AreaWidget() == area.area.QWidget, "area accessor is not wired to the underlying widget")
	proxy := area.FocusProxy()
	require.True(t, proxy != nil && proxy.UnsafePointer() == area.area.QWidget.UnsafePointer(),
		"wrapper focus proxy is not the text edit")
}

func TestRTextAreaBindTwoWay(t *testing.T) {
	model := reactive.NewSignal("a")
	area := NewRTextArea()
	area.Bind(model)
	require.Equal(t, "a", area.Text(), "adopts the bound state")

	area.SetText("b")
	require.Equal(t, "b", model.Peek(), "typing publishes back")

	model.Set("c")
	require.Equal(t, "c", area.Text(), "model changes push in")

	area.SetMonospace(true)
	require.Contains(t, area.area.StyleSheet(), "monospace", "monospace style was not applied")
}

// A rewriting model (e.g. KVEditor's JSON round-trip canonicalizing the text)
// used to echo into the widget on every valid keystroke: SetPlainText reset
// the cursor to the start. Our own edits must stay echoed out, not in.
func TestRTextAreaBindDoesNotEchoOwnEdits(t *testing.T) {
	v := reactive.NewSignal("")
	model := reactive.NewWritableComputed(
		func() string { return v.Get() },
		func(s string) { v.Set(s + "!") },
	)
	area := NewRTextArea()
	area.Bind(model)

	area.SetText("typed")
	require.Equal(t, "typed!", model.Peek(), "the edit still publishes through the model")
	require.Equal(t, "typed", area.Text(), "canonical form must not echo back over our own edit")

	// An external write is not an echo of our edit and still pushes in.
	v.Set("ext")
	require.Equal(t, "ext", area.Text())
}
