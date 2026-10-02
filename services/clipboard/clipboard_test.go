package clipboard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractURLVectors(t *testing.T) {
	clipboard := NewClipboard()
	assert.Equal(t, "https://example.com/a", clipboard.ExtractURL("get https://example.com/a now"))
	assert.Equal(t, "http://a", clipboard.ExtractURL("http://a then https://b"))
	assert.Empty(t, clipboard.ExtractURL("no links here"))
	assert.Empty(t, clipboard.ExtractURL(""))
}
