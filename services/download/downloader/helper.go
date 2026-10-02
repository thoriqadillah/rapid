package downloader

import (
	"mime"
	"net/url"
	"os/exec"
	"path"
	"rapid/lib"
	"strings"
	"syscall"
)

// getCategory guesses the lib.Category from mime and filename.
func getCategory(mimeType, fileName string) lib.Category {
	var guess string
	if fileName != "" {
		guess = mime.TypeByExtension(path.Ext(fileName))
	}

	if mimeType == "" || guess == "" {
		return lib.CategoryUnknown
	}

	if strings.HasPrefix(mimeType, "video/") || strings.HasPrefix(guess, "video/") {
		return lib.CategoryVideo
	}

	if strings.HasPrefix(mimeType, "audio/") || strings.HasPrefix(guess, "audio/") {
		return lib.CategoryAudio
	}

	if strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(guess, "image/") {
		return lib.CategoryImage
	}
	docs := map[string]bool{
		"application/pdf":  true,
		"application/json": true,
		"application/xml":  true,
	}

	if strings.HasPrefix(mimeType, "text/") || strings.HasPrefix(guess, "text/") || docs[mimeType] || docs[guess] {
		return lib.CategoryDocument
	}

	comp := map[string]bool{
		"application/zip":              true,
		"application/x-rar-compressed": true,
		"application/x-7z-compressed":  true,
		"application/gzip":             true,
		"application/x-tar":            true,
	}
	if comp[mimeType] || comp[guess] {
		return lib.CategoryCompressed
	}

	return lib.CategoryApplication
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

func isAlive(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}

	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		return false
	}

	return cmd.Process.Signal(syscall.Signal(0)) == nil
}
