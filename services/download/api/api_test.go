package api

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisplayName(t *testing.T) {
	cases := []struct {
		name string
		d    Download
		want string
	}{
		{
			"title wins",
			Download{
				GID:      "g1",
				Resolved: &ResolvedURL{Title: "Title", Filename: "file.bin"},
			},
			"Title",
		},
		{
			"filename fallback",
			Download{
				GID:      "g1",
				Resolved: &ResolvedURL{Filename: "file.bin"},
			},
			"file.bin",
		},
		{
			"basename fallback",
			Download{
				GID:   "g1",
				Files: []DownloadFile{{Path: "/a/b/c.mp4"}},
			},
			"c.mp4",
		},
		{
			"gid fallback",
			Download{GID: "g1"},
			"g1",
		},
	}
	for _, c := range cases {
		require.Equal(t, c.want, c.d.Name(), c.name)
	}
}

func TestPercentAndProgressText(t *testing.T) {
	d := Download{TotalLength: 3, CompletedLength: 1, DownloadSpeed: 10}
	require.Equal(t, "33%", d.FormatProgress())
	d = Download{TotalLength: 100, CompletedLength: 150}
	require.Equal(t, "150%", d.FormatProgress())
	require.Equal(t, "0%", Download{}.FormatProgress())
}

func TestEstText(t *testing.T) {
	cases := []struct {
		total, done, speed int64
		want               string
	}{
		{1000, 0, 100, "10s"},
		{10000, 0, 100, "1m"},
		{1000000, 0, 100, "2h 46m"},
		{100, 100, 100, "—"},
		{1000, 0, 0, "—"},
		{0, 0, 0, "—"},
	}
	for _, c := range cases {
		d := Download{TotalLength: c.total, CompletedLength: c.done, DownloadSpeed: c.speed}
		require.Equal(t, c.want, d.FormatEstimation(), "total=%d done=%d speed=%d", c.total, c.done, c.speed)
	}
}

func TestSpeedAndSizeText(t *testing.T) {
	d := Download{Status: "error", DownloadSpeed: 1024}
	require.Equal(t, "—", d.FormatSpeed())

	d = Download{Status: "active", DownloadSpeed: 2048}
	require.Equal(t, "2.0 KB/s", d.FormatSpeed())

	d = Download{Status: "paused"}
	require.Equal(t, "—", d.FormatSpeed())
	require.Equal(t, "—", Download{}.FormatSize())

	d = Download{TotalLength: 2048, CompletedLength: 1024}
	require.Equal(t, "1.0 KB / 2.0 KB", d.FormatSize())
}

func TestFileDir(t *testing.T) {
	withFile := Download{Dir: "/base", Files: []DownloadFile{{Path: "/downloads/sub/a.bin"}}}
	require.Equal(t, "/downloads/sub", withFile.FileDir())

	noFile := Download{Dir: "/base"}
	require.Equal(t, "/base", noFile.FileDir())
	require.Equal(t, "", Download{}.FileDir())
}

func TestStatusTextAndGuards(t *testing.T) {
	require.Equal(t, "Stopped", Download{Status: "removed"}.FormatStatus())
	require.Equal(t, "Active", Download{Status: "active"}.FormatStatus())
	require.Equal(t, "", Download{}.FormatStatus())

	require.True(t, Download{Status: "active"}.CanPause())
	require.True(t, Download{Status: "waiting"}.CanPause())
	require.False(t, Download{Status: "paused"}.CanPause())

	require.True(t, Download{Status: "paused"}.CanResume())
	require.False(t, Download{Status: "active"}.CanResume())

	require.True(t, Download{Status: "error"}.IsError())
	require.False(t, Download{Status: "complete"}.IsError())
}
