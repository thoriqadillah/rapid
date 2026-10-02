// Package lib holds small shared helpers. Check here before adding a new
// dependency or duplicating logic elsewhere.
package lib

import (
	"mime"
	"strconv"
	"strings"
)

func StringToInt[T int | int8 | int16 | int32 | int64](v string) T {
	i, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return T(i)
}

func StringToBool(v string) bool {
	b, _ := strconv.ParseBool(strings.TrimSpace(v))
	return b
}

func StringToFloat(v string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f
}

// canonicalExt prefers a stable extension for common types. The system mime
// table order is platform-dependent (e.g. .f4v sorts before .mp4 for
// video/mp4 on some distros), so guess only as a fallback.
var canonicalExt = map[string]string{
	"video/mp4":                    ".mp4",
	"audio/mpeg":                   ".mp3",
	"video/x-matroska":             ".mkv",
	"video/webm":                   ".webm",
	"audio/ogg":                    ".ogg",
	"audio/webm":                   ".weba",
	"image/webp":                   ".webp",
	"image/jpeg":                   ".jpg",
	"image/png":                    ".png",
	"image/gif":                    ".gif",
	"application/pdf":              ".pdf",
	"application/zip":              ".zip",
	"application/x-rar-compressed": ".rar",
	"application/x-7z-compressed":  ".7z",
	"application/gzip":             ".gz",
	"application/x-tar":            ".tar",
}

// ExtensionForType returns the preferred extension for a MIME type.
func ExtensionForType(m string) string {
	if e, ok := canonicalExt[m]; ok {
		return e
	}
	if exts, _ := mime.ExtensionsByType(m); len(exts) > 0 {
		return exts[0]
	}
	return ""
}

// IsAlnum reports whether s is non-empty ASCII alphanumeric.
func IsAlnum(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
