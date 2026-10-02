package clipboard

import "regexp"

// URLRegex matches the first http(s) URL in pasted text.
var URLRegex = regexp.MustCompile(`https?://\S+`)

type Clipboard struct{}

func NewClipboard() *Clipboard {
	return &Clipboard{}
}

func (c *Clipboard) ExtractURL(text string) string {
	return URLRegex.FindString(text)
}
