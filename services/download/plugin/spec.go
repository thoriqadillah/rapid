package plugin

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	liberrors "rapid/lib/errors"
	"rapid/lib/helpers/iter"
)

// ManifestName is the manifest file looked up in each plugin dir.
const ManifestName = "plugin.json"

// ResolverType is the only manifest type the app runs.
const ResolverType = "resolver"

// Manifest is a parsed plugin.json. matches holds the compiled patterns; a
// manifest with no patterns never matches. Dir anchors Command; S7 resolves
// the executable from these two.
type Manifest struct {
	ID      string
	Name    string
	Version string
	Dir     string
	Command string
	Args    []string
	matches []*regexp.Regexp
}

// LoadManifest reads dir/plugin.json. Decoding is strict: unknown fields
// ignored, id falls back to name, version to "0.0.0", non-list/non-string
// match and args fail the load. Anything unsafe or unusable (bad JSON,
// non-resolver type, uncompilable regex, escaping command) is an error
// wrapped in *errors.PluginError — identity is the manifest path, since id/name
// may be absent exactly when load fails; the S8 scan logs and skips it.
func LoadManifest(dir string) (Manifest, error) {
	manifest := filepath.Join(dir, ManifestName)

	raw, err := os.ReadFile(manifest)
	if err != nil {
		return Manifest{}, liberrors.NewPluginError(manifest, err)
	}

	var data struct {
		ID        string   `json:"id"`
		Name      string   `json:"name"`
		Version   string   `json:"version"`
		Type      string   `json:"type"`
		Match     []string `json:"match"`
		Transport struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"transport"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return Manifest{}, liberrors.NewPluginError(err.Error())
	}
	if data.Type != ResolverType {
		return Manifest{}, liberrors.PluginErrorf("type must be %q", ResolverType)
	}
	if data.Name == "" {
		return Manifest{}, liberrors.NewPluginError("missing name")
	}
	id := cmp.Or(data.ID, data.Name)
	version := cmp.Or(data.Version, "0.0.0")

	// Non-list match fails closed at decode (want []string); every pattern
	// compiles once here so a broken plugin fails at scan, not at resolve.
	// Empty patterns are rejected outright: an empty regex matches everything.
	matches, err := iter.MapErr(data.Match, func(m string) (*regexp.Regexp, error) {
		if m == "" {
			return nil, fmt.Errorf("empty match pattern")
		}
		return regexp.Compile(m)
	})
	if err != nil {
		return Manifest{}, liberrors.NewPluginError(err.Error())
	}

	command := data.Transport.Command
	if command == "" {
		return Manifest{}, liberrors.NewPluginError("transport.command is empty")
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return Manifest{}, liberrors.NewPluginError(err.Error())
	}
	if err := checkCommand(absDir, command); err != nil {
		return Manifest{}, liberrors.NewPluginError(err.Error())
	}

	return Manifest{
		ID:      id,
		Name:    data.Name,
		Version: version,
		Dir:     absDir,
		Command: command,
		Args:    data.Transport.Args,
		matches: matches,
	}, nil
}

// Matches reports whether any match regex hits uri (re.search semantics:
// unanchored). A manifest with no patterns never matches.
func (m Manifest) Matches(uri string) bool {
	for _, re := range m.matches {
		if re.MatchString(uri) {
			return true
		}
	}
	return false
}

// checkCommand rejects commands that would execute outside the plugin dir:
// absolute paths, and anything resolving above it via "..".
func checkCommand(dir, command string) error {
	if filepath.IsAbs(command) {
		return fmt.Errorf("transport.command must stay inside its dir, got absolute %q", command)
	}
	rel, err := filepath.Rel(dir, filepath.Join(dir, command))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("transport.command escapes its dir: %q", command)
	}
	return nil
}
