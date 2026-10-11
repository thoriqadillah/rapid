package settings

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// Paths are the platform locations the app uses.
type Paths struct {
	AppData   string
	Downloads string
}

// SystemPaths resolves Paths without Qt, matching Qt's
// AppDataLocation/DownloadLocation for application name "rapid".
// Linux AppData = $XDG_DATA_HOME/rapid else ~/.local/share/rapid;
// macOS/Windows = os.UserConfigDir()/rapid; Downloads honors
// ~/.config/user-dirs.dirs (XDG_DOWNLOAD_DIR) else ~/Downloads.
func SystemPaths() (Paths, error) {
	var p Paths

	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		cfg, err := os.UserConfigDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve app data dir: %w", err)
		}
		p.AppData = filepath.Join(cfg, "rapid")
	} else {
		// Qt on Linux: AppDataLocation = $XDG_DATA_HOME/<app> with XDG_DATA_HOME
		// defaulting to ~/.local/share (os.UserConfigDir would give ~/.config).
		if x := os.Getenv("XDG_DATA_HOME"); x != "" {
			p.AppData = filepath.Join(x, "rapid")
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return Paths{}, fmt.Errorf("resolve app data dir: %w", err)
			}
			p.AppData = filepath.Join(home, ".local", "share", "rapid")
		}
	}

	downloads, err := systemDownloadDir()
	if err != nil {
		return Paths{}, err
	}
	p.Downloads = downloads
	return p, nil
}

// Default builds settings rooted at baseDir, creating dirs as needed.
// Plugin dirs, in priority order: <baseDir>/plugins, <Paths.AppData>/plugins,
// then every entry of RAPID_PLUGIN_DIRS (later entries cannot override the
// bundled copy; the manager treats earlier dirs as lower priority).
func Default(baseDir string, p Paths) (Settings, error) {
	if p.AppData == "" || p.Downloads == "" {
		return Settings{}, errors.New("settings: app data and downloads paths must not be empty")
	}
	if err := os.MkdirAll(p.AppData, 0o755); err != nil {
		return Settings{}, fmt.Errorf("create app data dir: %w", err)
	}
	if err := os.MkdirAll(p.Downloads, 0o755); err != nil {
		return Settings{}, fmt.Errorf("create downloads dir: %w", err)
	}

	sessionFile := filepath.Join(p.AppData, "aria2.session")
	f, err := os.OpenFile(sessionFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return Settings{}, fmt.Errorf("create aria2 session file: %w", err)
	}
	f.Close()

	secret, err := loadOrCreateSecret(p.AppData)
	if err != nil {
		return Settings{}, fmt.Errorf("aria2 secret: %w", err)
	}

	pluginDirs := []string{
		filepath.Join(baseDir, "plugins"),
		filepath.Join(p.AppData, "plugins"),
	}
	if extra := os.Getenv("RAPID_PLUGIN_DIRS"); extra != "" {
		pluginDirs = append(pluginDirs, filepath.SplitList(extra)...)
	}

	return Settings{
		DataDir:                  p.AppData,
		DownloadDir:              p.Downloads,
		PluginDirs:               pluginDirs,
		BaseDir:                  baseDir,
		Aria2Host:                "127.0.0.1",
		Aria2Port:                6800,
		Aria2Token:               secret,
		Aria2SessionFile:         sessionFile,
		Aria2SaveSessionInterval: 1,
		PollIntervalMs:           1000,
	}, nil
}

// loadOrCreateSecret returns a persisted aria2 RPC secret, creating a random
// one on first run. It fails closed: without a secret, any local web page can
// POST to the aria2 RPC endpoint, so an unreadable secret store is an error
// rather than a silently unauthenticated daemon.
func loadOrCreateSecret(dir string) (string, error) {
	path := filepath.Join(dir, "aria2.secret")
	if b, err := os.ReadFile(path); err == nil {
		if s := strings.TrimSpace(string(b)); len(s) >= 16 {
			return s, nil
		}
	}
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	token := hex.EncodeToString(raw)
	if err := os.WriteFile(path, []byte(token), 0o600); err != nil {
		return "", fmt.Errorf("persist secret: %w", err)
	}
	return token, nil
}

func systemDownloadDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve downloads dir: %w", err)
	}
	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		if dir := xdgDownloadDir(home); dir != "" {
			return dir, nil
		}
	}
	return filepath.Join(home, "Downloads"), nil
}

// xdgDownloadDir reads XDG_DOWNLOAD_DIR from ~/.config/user-dirs.dirs,
// expanding $HOME like the XDG spec does.
func xdgDownloadDir(home string) string {
	b, err := os.ReadFile(filepath.Join(home, ".config", "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	for line := range strings.Lines(string(b)) {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "XDG_DOWNLOAD_DIR="); ok {
			dir := strings.TrimSpace(after)
			dir = strings.Trim(dir, `"'`)
			if dir == "" {
				return ""
			}
			return filepath.Clean(strings.ReplaceAll(dir, "$HOME", home))
		}
	}
	return ""
}
