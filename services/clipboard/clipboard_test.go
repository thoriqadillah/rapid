package clipboard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractURLVectors(t *testing.T) {
	clipboard := NewClipboard()
	require.Equal(t, "https://example.com/a", clipboard.ExtractURL("get https://example.com/a now"))
	require.Equal(t, "http://a", clipboard.ExtractURL("http://a then https://b"))
	require.Equal(t, "https://x.dev", clipboard.ExtractURL("see https://x.dev."))
	require.Equal(t, "ftp://files.example.com/a.zip", clipboard.ExtractURL("grab ftp://files.example.com/a.zip))"))
	require.Empty(t, clipboard.ExtractURL("no links here"))
	require.Empty(t, clipboard.ExtractURL(""))
}
