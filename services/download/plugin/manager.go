package plugin

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"rapid/services/download/api"
)

// Manager resolves URIs through external plugins. Immutable after
// construction, so no lock is needed.
type Manager struct {
	plugins []*PluginResolver
}

// NewManager scans each dir's child directories for plugin.json. Invalid
// manifests are logged and skipped; duplicate ids keep the first dir's plugin.
func NewManager(dirs []string) *Manager {
	m := &Manager{}
	seen := map[string]bool{}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // a missing plugin dir is normal
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			path := filepath.Join(dir, entry.Name())
			manifest, err := LoadManifest(path)
			if err != nil {
				slog.Warn("skip plugin", "path", path, "err", err)
				continue
			}
			if seen[manifest.ID] {
				continue
			}

			seen[manifest.ID] = true
			m.plugins = append(m.plugins, NewPluginResolver(manifest))
		}
	}

	return m
}

// Resolve runs the first plugin whose manifest matches uri. An error means the
// plugin claimed the URI and failed: surface it, never fall back. A nil result
// with no error means no plugin matched and the caller owns resolution.
func (m *Manager) Resolve(ctx context.Context, uri string, options map[string]any) ([]api.ResolvedURL, error) {
	for _, resolver := range m.plugins {
		if !resolver.ShouldResolve(uri) {
			continue
		}

		items, err := resolver.Resolve(ctx, uri, options)
		if err != nil {
			return nil, err
		}

		return items, nil
	}

	return nil, nil
}

// Len reports how many plugins were loaded.
func (m *Manager) Len() int {
	return len(m.plugins)
}
