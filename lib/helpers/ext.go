package helpers

import "mime"

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

// ExtensionForType returns the preferred extension for a MIME type. MIME
// parameters ("video/mp4; codecs=...") are stripped first, since the canonical
// table and mime.ExtensionsByType both key on the bare type.
func ExtensionForType(m string) string {
	if base, _, err := mime.ParseMediaType(m); err == nil {
		m = base
	}
	if e, ok := canonicalExt[m]; ok {
		return e
	}
	if exts, _ := mime.ExtensionsByType(m); len(exts) > 0 {
		return exts[0]
	}
	return ""
}
