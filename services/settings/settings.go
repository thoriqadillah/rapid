// Package settings holds the app settings, rooted at Qt standard locations
// like setting/models.py: app data under QStandardPaths AppDataLocation,
// downloads under DownloadLocation. This is the one backend package allowed
// to import miqt (paths must match what the Qt widgets layer resolves).
package settings

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	qt "github.com/mappu/miqt/qt6"
)

// Settings mirrors setting/models.py with plain strings (no Qt types).
type Settings struct {
	DataDir                  string
	DownloadDir              string
	PluginDirs               []string
	BaseDir                  string
	Aria2Host                string
	Aria2Port                int
	Aria2Token               string
	Aria2SessionFile         string
	Aria2SaveSessionInterval int
	PollIntervalMs           int
}

// Default builds settings rooted at baseDir, creating dirs as needed.
// Empty Qt locations (unknown platform paths) fall back to XDG-ish defaults.
func Default(baseDir string) Settings {
	appData := qt.QStandardPaths_WritableLocation(qt.QStandardPaths__AppDataLocation)
	if appData == "" {
		appData = filepath.Join(userCacheDir(), "rapid")
	}
	_ = os.MkdirAll(appData, 0o755)

	downloadDir := qt.QStandardPaths_WritableLocation(qt.QStandardPaths__DownloadLocation)
	if downloadDir == "" {
		downloadDir = fallbackDownloadDir()
	}

	sessionFile := filepath.Join(appData, "aria2.session")
	if f, err := os.OpenFile(sessionFile, os.O_CREATE|os.O_APPEND, 0o644); err == nil {
		_ = f.Close()
	}

	return Settings{
		DataDir:     appData,
		DownloadDir: downloadDir,
		PluginDirs: []string{
			filepath.Join(baseDir, "plugins"),
			filepath.Join(appData, "plugins"),
		},
		BaseDir:                  baseDir,
		Aria2Host:                "127.0.0.1",
		Aria2Port:                6800,
		Aria2Token:               loadOrCreateSecret(appData),
		Aria2SessionFile:         sessionFile,
		Aria2SaveSessionInterval: 1,
		PollIntervalMs:           1000,
	}
}

// loadOrCreateSecret returns a persisted aria2 RPC secret, creating a random
// one on first run. Without a secret, any local web page can POST to the aria2
// RPC endpoint (JSON top-level requests are CORS-preflight-free) and queue
// downloads; the file is reused so a restart can adopt the same daemon.
func loadOrCreateSecret(dir string) string {
	path := filepath.Join(dir, "aria2.secret")
	if b, err := os.ReadFile(path); err == nil {
		if s := strings.TrimSpace(string(b)); len(s) >= 16 {
			return s
		}
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return ""
	}
	token := hex.EncodeToString(raw)
	os.WriteFile(path, []byte(token), 0o600)
	return token
}

func fallbackDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, "Downloads")
}

func userCacheDir() string {
	if dir, err := os.UserCacheDir(); err == nil && dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".cache")
}
