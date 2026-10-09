package clipboard

import (
	"regexp"
	"strings"
)

var urlRegex = regexp.MustCompile(`(?i)\b(?:https?|ftps?)://[^\s<>"']+`)

type Clipboard struct{}

func NewClipboard() *Clipboard {
	return &Clipboard{}
}

func (c *Clipboard) ExtractURL(text string) string {
	return strings.TrimRight(urlRegex.FindString(text), ".,;:!?)]}")
}
