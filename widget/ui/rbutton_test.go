package ui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The churn guard must skip stylesheet re-application when nothing
// style-affecting changed, and re-apply when something did.
func TestRButtonStyleChurnGuard(t *testing.T) {
	b := NewRButton("Start", PrimaryVariant, false)
	b.setStyleSheet() // establish the style key

	b.QPushButton.SetStyleSheet("/*clobbered*/")
	b.setStyleSheet() // same state → guarded no-op
	require.Equal(t, "/*clobbered*/", b.StyleSheet(), "unchanged style key must not rewrite the stylesheet")

	b.SetLink(true) // state change → regenerates
	require.NotEqual(t, "/*clobbered*/", b.StyleSheet(), "changed style key must re-apply the stylesheet")
}
