package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeManifest(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ManifestName), []byte(body), 0o644))
	return dir
}

func TestLoadExample(t *testing.T) {
	m, err := LoadManifest("example")
	require.NoError(t, err)
	require.Equal(t, "sampledemo", m.ID)
	require.Equal(t, "resolver", ResolverType)
	require.True(t, m.Matches("https://www.youtube.com/watch?v=dQw4w9WgXcQ"))
	require.True(t, m.Matches("https://youtu.be/dQw4w9WgXcQ"))
	require.False(t, m.Matches("https://example.com/file.mp4"))
	require.Equal(t, "plugin.py", m.Command)
}

func TestLoadManifestValidation(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		check func(t *testing.T, m Manifest)
	}{
		{
			"minimal valid",
			`{
				"id": "x",
				"name": "X",
				"type": "resolver",
				"match": [],
				"transport":{
					"command": "p.py"
				}
			}`,
			func(t *testing.T, m Manifest) {
				require.Equal(t, "x", m.ID)
				require.Equal(t, "0.0.0", m.Version)
				require.False(t, m.Matches("https://anything.example/"))
			},
		},
		{
			"id falls back to name",
			`{
				"name": "X",
				"type": "resolver",
				"match": [],
				"transport": {
					"command": "p.py"
				}
			}`,
			func(t *testing.T, m Manifest) { require.Equal(t, "X", m.ID) },
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := LoadManifest(writeManifest(t, c.body))
			require.NoError(t, err)
			c.check(t, m)
		})
	}
}

func TestLoadManifestRejects(t *testing.T) {
	cases := []struct{ name, body string }{
		{"garbage json", `{oops`},
		{"non-object json", `["x"]`},
		{"missing type", `{"id":"x","name":"X","transport":{"command":"p.py"}}`},
		{"non-resolver type", `{"id":"x","name":"X","type":"daemon","transport":{"command":"p.py"}}`},
		{"missing name", `{"id":"x","type":"resolver","transport":{"command":"p.py"}}`},
		{"missing transport", `{"id":"x","name":"X","type":"resolver"}`},
		{"non-object transport", `{"id":"x","name":"X","type":"resolver","transport":"p.py"}`},
		{"empty command", `{"id":"x","name":"X","type":"resolver","transport":{"command":""}}`},
		{"non-string command", `{"id":"x","name":"X","type":"resolver","transport":{"command":42}}`},
		{"invalid regex", `{"id":"x","name":"X","type":"resolver","match":["(["],"transport":{"command":"p.py"}}`},
		{"non-string match entry", `{"id":"x","name":"X","type":"resolver","match":["a\\.example",42],"transport":{"command":"p.py"}}`},
		{"empty match entry", `{"id":"x","name":"X","type":"resolver","match":[""],"transport":{"command":"p.py"}}`},
		{"non-list match", `{"id":"x","name":"X","type":"resolver","match":"nope","transport":{"command":"p.py"}}`},
		{"absolute command", `{"id":"x","name":"X","type":"resolver","transport":{"command":"/bin/x"}}`},
		{"escaping command", `{"id":"x","name":"X","type":"resolver","transport":{"command":"../x"}}`},
		{"deep escaping command", `{"id":"x","name":"X","type":"resolver","transport":{"command":"sub/../../x"}}`},
		{"non-string args entry", `{"id":"x","name":"X","type":"resolver","transport":{"command":"p.py","args":["--port",8080]}}`},
		{"non-list args", `{"id":"x","name":"X","type":"resolver","transport":{"command":"p.py","args":"--port"}}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := LoadManifest(writeManifest(t, c.body))
			require.Error(t, err)
			var perr *PluginError
			require.ErrorAs(t, err, &perr, "load failures carry the plugin identity")
		})
	}
}

func TestLoadManifestMissingFile(t *testing.T) {
	_, err := LoadManifest(t.TempDir())
	require.Error(t, err)
}
