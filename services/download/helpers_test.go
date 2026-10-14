package download

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafeFileName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"plain.mkv", "plain.mkv"},
		{"a/b.mkv", "ab.mkv"},
		{`a\b.mkv`, "ab.mkv"},
		{"a\x00b.mkv", "ab.mkv"},
		{"../x", "..x"},
		{"/abs/path", "abspath"},
		{"", ""},
	}
	for _, c := range cases {
		require.Equal(t, c.want, safeFileName(c.in), "in=%q", c.in)
	}
}

func TestWithin(t *testing.T) {
	cases := []struct {
		dir, path string
		want      bool
	}{
		{"/dl", "/dl/a.mkv", true},
		{"/dl", "/dl/sub/a.mkv", true},
		{"/dl", "/dl2/a.mkv", false}, // shared prefix is not containment
		{"/dl", "/dl/../x", false},
		{"/dl", "/x", false},
		{"/dl", "/dl", true}, // the dir itself counts as within
		{"/dl/", "/dl/sub/../f", true},
	}
	for _, c := range cases {
		require.Equal(t, c.want, within(c.dir, c.path), "dir=%q path=%q", c.dir, c.path)
	}
}

func TestUniqueName(t *testing.T) {
	touch := func(t *testing.T, dir, name string) {
		t.Helper()
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644))
	}

	t.Run("no collision returns bare name", func(t *testing.T) {
		require.Equal(t, "a.mkv", uniqueName(t.TempDir(), "a.mkv"))
	})

	t.Run("collision appends counter", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "a.mkv")
		require.Equal(t, "a (1).mkv", uniqueName(dir, "a.mkv"))
	})

	t.Run("multiple collisions keep counting", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "a.mkv")
		touch(t, dir, "a (1).mkv")
		require.Equal(t, "a (2).mkv", uniqueName(dir, "a.mkv"))
	})

	t.Run("no-extension name", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "README")
		require.Equal(t, "README (1)", uniqueName(dir, "README"))
	})

	t.Run("aria2 control file counts as taken", func(t *testing.T) {
		dir := t.TempDir()
		touch(t, dir, "a.mkv.aria2")
		require.Equal(t, "a (1).mkv", uniqueName(dir, "a.mkv"))
	})
}
