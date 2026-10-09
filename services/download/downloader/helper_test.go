package downloader

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/lib"
	"rapid/services/download/api"
)

func TestGetCategory(t *testing.T) {
	cases := []struct {
		name string
		mime string
		file string
		want lib.Category
	}{
		{"mime only, no extension", "video/mp4", "download", lib.CategoryVideo},
		{"mime with parameters", "application/json; charset=utf-8", "", lib.CategoryDocument},
		{"extension only", "", "song.mp3", lib.CategoryAudio},
		{"both empty", "", "", lib.CategoryUnknown},
		{"known type, none of the buckets", "application/octet-stream", "", lib.CategoryApplication},
		{"zip", "application/zip", "", lib.CategoryCompressed},
		{"text", "text/plain", "", lib.CategoryDocument},
		{"image", "image/png", "", lib.CategoryImage},
	}
	for _, c := range cases {
		require.Equal(t, c.want, getCategory(c.mime, c.file), "%s: getCategory(%q, %q)", c.name, c.mime, c.file)
	}
}

// A referer/cookies with no explicit headers used to write into a nil map
// (maps.Clone(nil) == nil) and panic.
func TestAria2OptionsHandlesNilHeaders(t *testing.T) {
	d := &Aria2Downloader{}
	opts := d.aria2Options(api.ResolvedURL{
		URL:     "https://example.com/file.mp4",
		Referer: "https://example.com/page",
		Cookies: map[string]string{"session": "abc"},
	})

	lines, ok := opts["header"].([]string)
	require.True(t, ok && len(lines) > 0, "expected header lines, got %#v", opts["header"])
	joined := strings.Join(lines, "\n")
	require.Contains(t, joined, "Referer: https://example.com/page", "Referer missing from headers")
	require.Contains(t, joined, "Cookie: session=abc", "Cookie missing from headers")
}
