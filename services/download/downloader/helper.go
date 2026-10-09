package downloader

import (
	"mime"
	"net/url"
	"path"
	"rapid/lib"
	"strings"
)

var (
	docTypes = map[string]bool{
		"application/pdf":  true,
		"application/json": true,
		"application/xml":  true,
	}
	zipTypes = map[string]bool{
		"application/zip":              true,
		"application/x-rar-compressed": true,
		"application/x-7z-compressed":  true,
		"application/gzip":             true,
		"application/x-tar":            true,
	}
)

// normalizeMIME strips parameters ("video/mp4; codecs=..." -> "video/mp4") and
// returns "" for anything unparseable.
func normalizeMIME(s string) string {
	if s == "" {
		return ""
	}
	mt, _, err := mime.ParseMediaType(s)
	if err != nil {
		return ""
	}
	return mt
}

func categoryOfMIME(t string) lib.Category {
	switch {
	case strings.HasPrefix(t, "video/"):
		return lib.CategoryVideo
	case strings.HasPrefix(t, "audio/"):
		return lib.CategoryAudio
	case strings.HasPrefix(t, "image/"):
		return lib.CategoryImage
	case strings.HasPrefix(t, "text/"), docTypes[t]:
		return lib.CategoryDocument
	case zipTypes[t]:
		return lib.CategoryCompressed
	}
	return lib.CategoryUnknown
}

// getCategory guesses the lib.Category from mime and filename.
//
// Either source alone is enough: a server that sends "video/mp4" for a URL
// without a file extension (very common, e.g. "/download?id=1") must still be
// classified as video. Parameters are stripped first.
func getCategory(mimeType, fileName string) lib.Category {
	candidates := []string{normalizeMIME(mimeType)}
	if fileName != "" {
		candidates = append(candidates, normalizeMIME(mime.TypeByExtension(path.Ext(fileName))))
	}

	known := false
	for _, t := range candidates {
		if t == "" {
			continue
		}
		known = true
		if c := categoryOfMIME(t); c != lib.CategoryUnknown {
			return c
		}
	}
	if known {
		// A recognised type that is none of the above (e.g. application/octet-stream).
		return lib.CategoryApplication
	}
	return lib.CategoryUnknown
}

// filenameOf returns the URL path basename, or "" when absent.
func filenameOf(rawURI string) string {
	u, err := url.Parse(rawURI)
	if err != nil || u.Path == "" {
		return ""
	}
	name := path.Base(u.Path)
	if name == "" || name == "/" || name == "." {
		return ""
	}
	return name
}
