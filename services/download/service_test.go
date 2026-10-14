package download

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/db"
	"rapid/lib"
	"rapid/lib/reactive"
	"rapid/services/download/api"
	"rapid/services/download/downloader"
	"rapid/services/settings"
)

func newServiceWithItems(items []api.Download) *Service {
	s := NewService(settings.Settings{})
	s.items.Set(items)
	return s
}

// fakeDownloader is a tiny in-memory Downloader for service tests. Tests
// never boot a daemon; S4/S5/S9 extend it with scripted action behaviors.
type fakeDownloader struct {
	started    bool
	stopped    bool
	known      map[string]api.Download // gids GetStatus knows
	listened   []string
	unlistened []string
}

var _ downloader.Downloader = (*fakeDownloader)(nil)

func (f *fakeDownloader) Start(context.Context) error {
	f.started = true
	return nil
}

func (f *fakeDownloader) Stop() error {
	f.stopped = true
	return nil
}

func (f *fakeDownloader) Download(context.Context, api.ResolvedURL) (api.Download, error) {
	return api.Download{}, nil
}

func (f *fakeDownloader) Pause(context.Context, string) (api.Download, error) {
	return api.Download{}, nil
}

func (f *fakeDownloader) Resume(context.Context, string) (api.Download, error) {
	return api.Download{}, nil
}

func (f *fakeDownloader) Remove(context.Context, string) (api.Download, error) {
	return api.Download{}, nil
}

func (f *fakeDownloader) Purge(context.Context, string) error {
	return nil
}

func (f *fakeDownloader) GetStatus(_ context.Context, id string) (api.Download, error) {
	if d, ok := f.known[id]; ok {
		return d, nil
	}
	return api.Download{}, fmt.Errorf("unknown gid %q", id)
}

func (f *fakeDownloader) Listen(id string, _ downloader.NotifyCallback, _ downloader.ErrorCallback) {
	f.listened = append(f.listened, id)
}

func (f *fakeDownloader) Unlisten(id string) {
	f.unlistened = append(f.unlistened, id)
}

func (f *fakeDownloader) Refresh(context.Context, ...string) {}

func TestServiceFilterAndCounts(t *testing.T) {
	s := newServiceWithItems([]api.Download{
		{GID: "v1", Category: lib.CategoryVideo, Resolved: &api.ResolvedURL{Title: "Interstellar"}},
		{GID: "v2", Category: lib.CategoryVideo, Resolved: &api.ResolvedURL{Title: "Dune"}},
		{GID: "a1", Category: lib.CategoryAudio, Files: []api.DownloadFile{{Path: "/music/album.zip"}}},
		{GID: "u1"},
	})

	counts := s.Counts.Get()
	require.Equal(t, 4, counts["all"])
	require.Equal(t, 2, counts["video"])
	require.Equal(t, 1, counts["audio"])
	require.NotContains(t, counts, "")

	s.Category.Set("video")
	require.Len(t, s.Items.Get(), 2)

	// The sidebar "All downloads" destination is "all": selecting it after a
	// category must show everything again (matches treats "all" as all).
	s.Category.Set("all")
	require.Len(t, s.Items.Get(), 4)
	require.Equal(t, "all", s.Category.Peek())

	s.Search.Set("dune")
	filtered := s.Items.Get()
	require.Len(t, filtered, 1)
	require.Equal(t, "v2", filtered[0].GID)

	// Clearing the category keeps the search.
	s.Category.Set("")
	require.Len(t, s.Items.Get(), 1)

	s.Search.Set("  DUNE  ")
	require.Len(t, s.Items.Get(), 1)
	require.Equal(t, "  DUNE  ", s.Search.Peek())

	s.Search.Set("MP4")
	mp4Upper := len(s.Items.Get())
	s.Search.Set("mp4")
	require.Equal(t, mp4Upper, len(s.Items.Get()), "MP4 and mp4 must match the same rows")
	require.Equal(t, "mp4", s.Search.Peek())
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
		{"uppercase search", video, "", "STELLAR", true},
		{"padded search", video, "", "  stellar  ", true},
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

func TestFilterChangedOnlyTracksFilters(t *testing.T) {
	s := newServiceWithItems([]api.Download{{GID: "x", Status: "active"}})
	var runs int
	dispose := reactive.Effect(func() {
		s.FilterChanged.Get()
		runs++
	})
	defer dispose()
	runs = 0 // ignore the initial dependency-collecting run

	// A progress tick changes items but must not collapse the expanded row.
	s.items.Set([]api.Download{{GID: "x", Status: "active", DownloadSpeed: 99}})
	require.Equal(t, 0, runs, "progress tick must not fire FilterChanged")

	s.Search.Set("foo")
	require.Equal(t, 1, runs)
	s.Category.Set("video")
	require.Equal(t, 2, runs)
}

func TestServiceOnChanged(t *testing.T) {
	s := newServiceWithItems(nil)
	calls := -1 // ignore the initial dependency-collecting run
	unsubscribe := reactive.Effect(func() {
		s.Items.Get()
		s.Counts.Get()
		calls++
	})
	defer unsubscribe()

	s.Category.Set("video")
	s.Search.Set("abc")
	s.items.Set([]api.Download{{GID: "x"}})
	require.Equal(t, 3, calls)

	// Idempotent updates still count as changes? No: same filter is a no-op.
	s.Category.Set("video")
	s.Search.Set("abc")
	require.Equal(t, 3, calls)
}

func TestServiceUnsubscribe(t *testing.T) {
	s := NewService(settings.Settings{})
	var calls int
	unsubscribe := reactive.Effect((func() { calls++ }))

	s.items.Set([]api.Download{{GID: "x"}})
	require.Equal(t, 1, calls)

	unsubscribe()
	s.items.Set([]api.Download{{GID: "y"}})
	require.Equal(t, 1, calls, "unsubscribed callback must not fire")

	// Unsubscribing twice is safe.
	unsubscribe()
}

func TestServiceStartLoadsAndClose(t *testing.T) {
	require.NoError(t, db.OpenMemory(t.Context()))
	t.Cleanup(func() { db.Close() })
	mustUpsert(t, db.DB(), api.Download{GID: "a", Status: "waiting", Category: lib.CategoryVideo})
	mustUpsert(t, db.DB(), api.Download{GID: "b", Category: lib.CategoryAudio})

	s := NewService(settings.Settings{})
	fake := &fakeDownloader{known: map[string]api.Download{
		"a": {GID: "a", Status: "waiting"},
	}}
	s.downloader = fake
	require.NoError(t, s.Start(t.Context()))

	all := s.items.Peek()
	require.Len(t, all, 2)
	require.Equal(t, "b", all[0].GID, "newest-first order")
	require.Equal(t, "a", all[1].GID)
	require.Equal(t, 2, s.Counts.Get()["all"])

	// Only the waiting row is re-listened; "b" is idle and the daemon is
	// asked about nothing else.
	require.True(t, fake.started)
	require.Equal(t, []string{"a"}, fake.listened)

	s.Close()
	require.True(t, fake.stopped)
	s.Close() // idempotent
}

// newMergeService seeds both the DB and the in-memory slice with items;
// merge tests call s.mergeDownload directly (no Qt event loop needed).
func newMergeService(t *testing.T, items []api.Download) *Service {
	t.Helper()
	require.NoError(t, db.OpenMemory(t.Context()))
	t.Cleanup(func() { db.Close() })
	for _, d := range items {
		mustUpsert(t, db.DB(), d)
	}
	s := NewService(settings.Settings{})
	s.items.Set(cloneDownloads(items))
	return s
}

func TestMergePreservesResolvedAndCategory(t *testing.T) {
	prev := api.Download{GID: "a", Status: "active", Category: lib.CategoryVideo,
		Resolved:        &api.ResolvedURL{URL: "https://cdn.example.com/dune.mkv", Title: "Dune", Filename: "dune.mkv"},
		DownloadSpeed:   100,
		TotalLength:     1000,
		CompletedLength: 100,
		Files:           []api.DownloadFile{{Index: 0, Path: "/dl/dune.mkv"}},
	}
	s := newMergeService(t, []api.Download{prev})

	s.upsertDownload(api.Download{GID: "a", Status: "active", DownloadSpeed: 200,
		TotalLength: 1000, CompletedLength: 200,
		Files: []api.DownloadFile{{Index: 0, Path: "/dl/dune.mkv"}}})

	got := s.items.Peek()[0]
	require.Equal(t, "Dune", got.Resolved.Title, "aria2 never reports resolved metadata")
	require.Equal(t, lib.CategoryVideo, got.Category)
	require.Equal(t, int64(200), got.DownloadSpeed)

	dbRow, err := getDownload(t.Context(), db.DB(), "a")
	require.NoError(t, err)
	require.Equal(t, "Dune", dbRow.Resolved.Title, "preserved fields persist")
}

func TestMergeActiveSpeedZeroIsAuthoritative(t *testing.T) {
	prev := api.Download{GID: "a", Status: "active", DownloadSpeed: 100, TotalLength: 1000, CompletedLength: 100}
	s := newMergeService(t, []api.Download{prev})

	s.upsertDownload(api.Download{GID: "a", Status: "active", DownloadSpeed: 0, TotalLength: 1000, CompletedLength: 100})
	require.Equal(t, int64(0), s.items.Peek()[0].DownloadSpeed,
		"stalled download must read 0, not a stale number")

	hist, err := getSpeedHistory(t.Context(), db.DB(), "a", 60)
	require.NoError(t, err)
	require.Empty(t, hist, "no sample for a zero-speed tick")
}

func TestMergeTransitionsFireOnce(t *testing.T) {
	s := newMergeService(t, []api.Download{
		{GID: "c", Status: "active"},
		{GID: "e", Status: "waiting"},
	})
	var completed, failed []string
	s.OnCompleted(func(d api.Download) { completed = append(completed, d.GID) })
	drop := s.OnCompleted(func(d api.Download) { completed = append(completed, "dropped:"+d.GID) })
	drop()
	s.OnError(func(err error) { failed = append(failed, err.Error()) })

	s.upsertDownload(api.Download{GID: "c", Status: "complete"})
	s.upsertDownload(api.Download{GID: "c", Status: "complete"}) // redelivery: silent
	s.upsertDownload(api.Download{GID: "e", Status: "error", ErrorMessage: "boom"})
	s.upsertDownload(api.Download{GID: "e", Status: "error", ErrorMessage: "boom"})

	require.Equal(t, []string{"c"}, completed)
	require.Equal(t, []string{"e: boom"}, failed)

	// An unknown gid arriving terminal still fires (no previous state).
	s.upsertDownload(api.Download{GID: "new", Status: "complete"})
	require.Equal(t, []string{"c", "new"}, completed)
}

func TestMergePrependsUnknownGidAndSamplesSpeed(t *testing.T) {
	s := newMergeService(t, []api.Download{{GID: "b", Status: "paused"}})

	s.upsertDownload(api.Download{GID: "a", Status: "active", DownloadSpeed: 512,
		TotalLength: 1000, CompletedLength: 10})

	peek := s.items.Peek()
	require.Len(t, peek, 2)
	require.Equal(t, "a", peek[0].GID, "new gids prepend newest-first")

	hist, err := getSpeedHistory(t.Context(), db.DB(), "a", 60)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	require.Equal(t, int64(512), hist[0].Speed)

	dbRow, err := getDownload(t.Context(), db.DB(), "a")
	require.NoError(t, err)
	require.Equal(t, "a", dbRow.GID)
}
