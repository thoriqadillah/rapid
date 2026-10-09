package download

import (
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/db"
	"rapid/lib"
	"rapid/services/download/api"
)

func newServiceWithItems(items []api.Download) *Service {
	return &Service{items: cloneDownloads(items)}
}

func TestServiceFilterAndCounts(t *testing.T) {
	s := newServiceWithItems([]api.Download{
		{GID: "v1", Category: lib.CategoryVideo, Resolved: &api.ResolvedURL{Title: "Interstellar"}},
		{GID: "v2", Category: lib.CategoryVideo, Resolved: &api.ResolvedURL{Title: "Dune"}},
		{GID: "a1", Category: lib.CategoryAudio, Files: []api.DownloadFile{{Path: "/music/album.zip"}}},
		{GID: "u1"},
	})

	counts := s.Counts()
	require.Equal(t, 4, counts["all"])
	require.Equal(t, 2, counts["video"])
	require.Equal(t, 1, counts["audio"])
	require.NotContains(t, counts, "")

	s.SetCategory("video")
	require.Len(t, s.ComputedItems(), 2)

	// The sidebar "All downloads" destination is "all": selecting it after a
	// category must show everything again.
	s.SetCategory("all")
	require.Len(t, s.ComputedItems(), 4)
	require.Empty(t, s.Category())

	s.SetSearch("dune")
	filtered := s.ComputedItems()
	require.Len(t, filtered, 1)
	require.Equal(t, "v2", filtered[0].GID)

	// Clearing the category keeps the search.
	s.SetCategory("")
	require.Len(t, s.ComputedItems(), 1)

	s.SetSearch("  DUNE  ")
	require.Len(t, s.ComputedItems(), 1)
	require.Equal(t, "dune", s.Search())
}

func TestServiceMatches(t *testing.T) {
	video := api.Download{
		Category: lib.CategoryVideo,
		Resolved: &api.ResolvedURL{Title: "Interstellar", Filename: "interstellar.mkv"},
		Files:    []api.DownloadFile{{Path: "/movies/interstellar.mkv"}},
	}
	s := newServiceWithItems(nil)
	cases := []struct {
		name     string
		d        api.Download
		category string
		search   string
		want     bool
	}{
		{"empty category matches all", video, "", "", true},
		{"all matches all", video, "all", "", true},
		{"exact category", video, "video", "", true},
		{"wrong category", video, "audio", "", false},
		{"title hit", video, "", "stellar", true},
		{"filename fallback", api.Download{
			Category: lib.CategoryVideo,
			Resolved: &api.ResolvedURL{Filename: "dune.mkv"},
		}, "", "dune", true},
		{"basename fallback", api.Download{
			Files: []api.DownloadFile{{Path: "/music/album.zip"}},
		}, "", "album", true},
		{"miss", video, "", "tenet", false},
		{"category mismatch blocks title hit", video, "audio", "stellar", false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, s.matches(c.d, c.category, c.search), c.name)
	}
}

func TestServiceOnChanged(t *testing.T) {
	s := newServiceWithItems(nil)
	var calls int
	s.OnChanged(func() { calls++ })

	s.SetCategory("video")
	s.SetSearch("abc")
	s.SetItems([]api.Download{{GID: "x"}})
	require.Equal(t, 3, calls)

	// Idempotent updates still count as changes? No: same filter is a no-op.
	s.SetCategory("video")
	s.SetSearch("abc")
	require.Equal(t, 3, calls)
}

func TestServiceUnsubscribeAndNoOpRefresh(t *testing.T) {
	require.NoError(t, db.OpenMemory(t.Context()))
	t.Cleanup(func() { db.Close() })
	mustUpsert(t, db.DB(), api.Download{GID: "a", Category: lib.CategoryVideo})

	s := NewService()
	var calls int
	unsubscribe := s.OnChanged(func() { calls++ })

	require.NoError(t, s.Refresh(t.Context()))
	require.Equal(t, 1, calls, "first load should notify")

	// The widget layer polls Refresh on a timer: an unchanged result must be a
	// no-op so the UI does not rebuild (and re-allocate Qt objects) every tick.
	require.NoError(t, s.Refresh(t.Context()))
	require.Equal(t, 1, calls, "unchanged refresh must not notify")

	mustUpsert(t, db.DB(), api.Download{GID: "b", Category: lib.CategoryAudio})
	require.NoError(t, s.Refresh(t.Context()))
	require.Equal(t, 2, calls, "changed refresh should notify")

	unsubscribe()
	s.SetCategory("video")
	require.Equal(t, 2, calls, "unsubscribed callback must not fire")

	// Unsubscribing twice is safe.
	unsubscribe()
}

func TestServiceRefreshFromDB(t *testing.T) {
	require.NoError(t, db.OpenMemory(t.Context()))
	t.Cleanup(func() { db.Close() })

	mustUpsert(t, db.DB(), api.Download{GID: "a", Category: lib.CategoryVideo})
	mustUpsert(t, db.DB(), api.Download{GID: "b", Category: lib.CategoryAudio})

	s := NewService()
	require.NoError(t, s.Refresh(t.Context()))

	all := s.All()
	require.Len(t, all, 2)
	require.Equal(t, "b", all[0].GID, "newest-first order")
	require.Equal(t, "a", all[1].GID)

	require.Equal(t, 2, s.Counts()["all"])
	require.Equal(t, "b", s.ComputedItems()[0].GID)
}
