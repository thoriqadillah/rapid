package download

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"rapid/services/download/api"
)

// safeFileName strips path separators and NULs so a plugin/title-derived
// name can never escape the download dir.
func safeFileName(s string) string {
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "\\", "")
	return strings.ReplaceAll(s, "\x00", "")
}

// within reports whether path stays inside dir. The dir itself counts as
// within; anything resolving to ".." or above does not.
func within(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// uniqueName returns name unless dir/name (or its aria2 control file) is
// taken, else the first free "name (i).ext".
func uniqueName(dir, name string) string {
	if !taken(dir, name) {
		return name
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		if cand := fmt.Sprintf("%s (%d)%s", stem, i, ext); !taken(dir, cand) {
			return cand
		}
	}
}

func taken(dir, name string) bool {
	return fileExists(filepath.Join(dir, name)) || fileExists(filepath.Join(dir, name+".aria2"))
}

// removeFiles deletes every path and its aria2 control file, ignoring
// missing files.
func removeFiles(files []api.DownloadFile) {
	for _, f := range files {
		if f.Path == "" {
			continue
		}
		for _, p := range []string{f.Path, f.Path + ".aria2"} {
			if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
				slog.Warn("download: remove file", "path", p, "err", err)
			}
		}
	}
}

// fileExists mirrors os.path.exists: any stat error means missing.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func cloneItems(items []api.Download) []api.Download {
	if items == nil {
		return nil
	}
	out := make([]api.Download, len(items))
	copy(out, items)
	return out
}
