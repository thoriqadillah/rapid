package download

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"rapid/db"
	"rapid/lib"
	"rapid/lib/errors"
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
	started     bool
	stopped     bool
	known       map[string]api.Download // gids GetStatus knows
	listened    []string
	unlistened  []string
	calls       []string // "op:gid" in execution order
	pauseErr    error
	resumeErr   error
	removeErr   error
	purgeErr    error
	downloadErr map[string]error
	downloads   []api.ResolvedURL
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

func (f *fakeDownloader) Download(_ context.Context, uri api.ResolvedURL) (api.Download, error) {
	f.calls = append(f.calls, "download:"+uri.URL)
	f.downloads = append(f.downloads, uri)
	if err := f.downloadErr[uri.URL]; err != nil {
		return api.Download{}, err
	}
	return api.Download{GID: fmt.Sprintf("g%d", len(f.downloads)), Status: "active", Dir: uri.Dir}, nil
}

func (f *fakeDownloader) Pause(_ context.Context, id string) (api.Download, error) {
	f.calls = append(f.calls, "pause:"+id)
	if f.pauseErr != nil {
		return api.Download{}, f.pauseErr
	}
	d, ok := f.known[id]
	if !ok {
		return api.Download{}, fmt.Errorf("unknown gid %q", id)
	}
	d.Status = "paused"
	f.known[id] = d
	return d, nil
}

func (f *fakeDownloader) Resume(_ context.Context, id string) (api.Download, error) {
	f.calls = append(f.calls, "resume:"+id)
	if f.resumeErr != nil {
		return api.Download{}, f.resumeErr
	}
	d, ok := f.known[id]
	if !ok {
		return api.Download{}, fmt.Errorf("unknown gid %q", id)
	}
	d.Status = "active"
	f.known[id] = d
	return d, nil
}

func (f *fakeDownloader) Remove(_ context.Context, id string) (api.Download, error) {
	f.calls = append(f.calls, "remove:"+id)
	if f.removeErr != nil {
		return api.Download{}, f.removeErr
	}
	d, ok := f.known[id]
	if !ok {
		return api.Download{}, fmt.Errorf("unknown gid %q", id)
	}
	d.Status = "removed"
	f.known[id] = d
	return d, nil
}

func (f *fakeDownloader) Purge(_ context.Context, id string) error {
	f.calls = append(f.calls, "purge:"+id)
	return f.purgeErr
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

	counts := s.Counts.Peek()
	require.Equal(t, 4, counts["all"])
	require.Equal(t, 2, counts["video"])
	require.Equal(t, 1, counts["audio"])
	require.NotContains(t, counts, "")

	s.Category.Set("video")
	require.Len(t, s.Items.Peek(), 2)

	// The sidebar "All downloads" destination is "all": selecting it after a
	// category must show everything again (matches treats "all" as all).
	s.Category.Set("all")
	require.Len(t, s.Items.Peek(), 4)
	require.Equal(t, "all", s.Category.Peek())

	s.Search.Set("dune")
	filtered := s.Items.Peek()
	require.Len(t, filtered, 1)
	require.Equal(t, "v2", filtered[0].GID)

	// Clearing the category keeps the search.
	s.Category.Set("")
	require.Len(t, s.Items.Peek(), 1)

	s.Search.Set("  DUNE  ")
	require.Len(t, s.Items.Peek(), 1)
	require.Equal(t, "  DUNE  ", s.Search.Peek())

	s.Search.Set("MP4")
	mp4Upper := len(s.Items.Peek())
	s.Search.Set("mp4")
	require.Equal(t, mp4Upper, len(s.Items.Peek()), "MP4 and mp4 must match the same rows")
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
	require.Equal(t, 2, s.Counts.Peek()["all"])

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
	s.items.Set(items)
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
	require.Equal(t, []string{"Failed to download e: boom"}, failed)

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

// newActionService builds a live service around a fake downloader: Start
// boots the real queue (t.Context cancels it at test end) with no DB rows,
// then memory is seeded. No Qt loop runs here, so GUI-marshaled tails are
// dropped — tests assert downloader-side effects plus direct GUI-method
// checks.
func newActionService(t *testing.T, items []api.Download) (*Service, *fakeDownloader) {
	t.Helper()
	require.NoError(t, db.OpenMemory(t.Context()))
	t.Cleanup(func() { db.Close() })
	s := NewService(settings.Settings{})
	fake := &fakeDownloader{known: map[string]api.Download{}}
	for _, d := range items {
		fake.known[d.GID] = d
	}
	s.downloader = fake
	require.NoError(t, s.Start(t.Context()))
	t.Cleanup(s.Close)
	s.items.Set(items)
	return s, fake
}

// syncQueue enqueues a barrier behind every pending op; the worker drains in
// order, so returning means earlier actions finished their downloader work.
func syncQueue(t *testing.T, s *Service) {
	t.Helper()
	done := make(chan struct{})
	if !s.queue.Go(func(context.Context) { close(done) }) {
		t.Fatal("action queue closed")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for action queue")
	}
}

func TestPauseResumePreserveOrder(t *testing.T) {
	s, fake := newActionService(t, []api.Download{{GID: "a", Status: "active"}})
	s.Pause(t.Context(), "a")
	s.Resume(t.Context(), "a")
	syncQueue(t, s)
	require.Equal(t, []string{"pause:a", "resume:a"}, fake.calls)
	require.Equal(t, []string{"a"}, fake.unlistened, "pause stops listening")
	require.Equal(t, []string{"a"}, fake.listened, "resume listens again")
}

func TestPauseFailureKeepsRow(t *testing.T) {
	s, fake := newActionService(t, []api.Download{{GID: "a", Status: "active"}})
	fake.pauseErr = fmt.Errorf("daemon gone")
	s.Pause(t.Context(), "a")
	syncQueue(t, s)
	require.Equal(t, []string{"pause:a"}, fake.calls)
	require.Empty(t, fake.unlistened, "failed pause keeps listening")
	require.Equal(t, "active", s.items.Peek()[0].Status, "row keeps its state")
}

func TestStopRetainsRemovedRow(t *testing.T) {
	s, fake := newActionService(t, []api.Download{{GID: "a", Status: "active"}})
	s.Stop(t.Context(), "a")
	syncQueue(t, s)
	require.Equal(t, []string{"remove:a"}, fake.calls)
	require.Equal(t, []string{"a"}, fake.unlistened)
	require.Equal(t, "removed", fake.known["a"].Status)
}

func TestDeleteUnlinksFilesAndPurges(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.mkv")
	aria2 := file + ".aria2"
	keep := filepath.Join(dir, "keep.mkv")
	for _, p := range []string{file, aria2, keep} {
		require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
	}
	s, fake := newActionService(t, []api.Download{{
		GID: "a", Status: "complete",
		Files: []api.DownloadFile{{Index: 0, Path: file}},
	}})
	s.Delete(t.Context(), "a", true)
	syncQueue(t, s)
	require.Equal(t, []string{"remove:a", "purge:a"}, fake.calls)
	require.Equal(t, []string{"a"}, fake.unlistened)
	require.NoFileExists(t, file)
	require.NoFileExists(t, aria2)
	require.FileExists(t, keep)
}

func TestDeleteNotFoundProceeds(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.mkv")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	s, fake := newActionService(t, []api.Download{{
		GID: "a", Status: "error",
		Files: []api.DownloadFile{{Index: 0, Path: file}},
	}})
	fake.removeErr = errors.Aria2Errorf("aria2 error code 1: GID#a is not found")
	fake.purgeErr = errors.Aria2Errorf("aria2 error code 1: GID#a is not found")
	s.Delete(t.Context(), "a", true)
	syncQueue(t, s)
	require.NoFileExists(t, file, "unknown gid counts as already gone")
	require.Equal(t, []string{"a"}, fake.unlistened)
}

func TestDeleteKeepsRowOnRemoveError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.mkv")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	s, fake := newActionService(t, []api.Download{{
		GID: "a", Status: "active",
		Files: []api.DownloadFile{{Index: 0, Path: file}},
	}})
	fake.removeErr = fmt.Errorf("connection refused")
	s.Delete(t.Context(), "a", true)
	syncQueue(t, s)
	require.FileExists(t, file, "aborted delete touches nothing")
	require.Empty(t, fake.unlistened)
	require.Equal(t, "active", s.items.Peek()[0].Status)
}

func TestActiveCount(t *testing.T) {
	s := NewService(settings.Settings{})
	s.items.Set([]api.Download{
		{GID: "a", Status: "active"},
		{GID: "b", Status: "paused"},
		{GID: "c", Status: "complete"},
		{GID: "d", Status: "error"},
		{GID: "e", Status: "waiting"},
		{GID: "f", Status: "removed"},
	})
	require.Equal(t, 1, s.ActiveCount.Peek(), "only transferring rows count")

	var runs int
	dispose := reactive.Effect(func() { s.ActiveCount.Get(); runs++ })
	defer dispose()
	s.items.Set(s.items.Peek())
	require.Equal(t, 1, runs, "same count must not notify")
}

func TestMergeUpdatesActiveCount(t *testing.T) {
	s := newMergeService(t, []api.Download{{GID: "a", Status: "active"}})
	require.Equal(t, 1, s.ActiveCount.Peek())
	s.upsertDownload(api.Download{GID: "a", Status: "complete"})
	require.Equal(t, 0, s.ActiveCount.Peek())
}

func TestMergeSkipsStaleEcho(t *testing.T) {
	s := newMergeService(t, []api.Download{{
		GID:             "a",
		Status:          "active",
		DownloadSpeed:   100,
		TotalLength:     1000,
		CompletedLength: 100,
	}})
	s.settings.PollIntervalMs = 1000
	now := time.Now()

	s.stampAction("a") // user pauses; in-memory state is still active

	// A straggler carrying the pre-action state is skipped...
	s.upsertDownload(api.Download{
		GID:             "a",
		Status:          "active",
		DownloadSpeed:   50,
		TotalLength:     1000,
		CompletedLength: 50,
	})
	require.Equal(t, int64(100), s.items.Peek()[0].DownloadSpeed)
	dbRow, err := getDownload(t.Context(), db.DB(), "a")
	require.NoError(t, err)
	require.Equal(t, int64(100), dbRow.DownloadSpeed, "skipped echo writes nothing")

	// ...while the confirming update applies and clears the stamp.
	s.upsertDownload(api.Download{GID: "a", Status: "paused"})
	require.Equal(t, "paused", s.items.Peek()[0].Status)

	// After the window, same-state notifies apply normally again.
	s.stampAction("a")
	now = now.Add(3 * time.Second)
	s.upsertDownload(api.Download{
		GID:             "a",
		Status:          "paused",
		DownloadSpeed:   9,
		TotalLength:     1000,
		CompletedLength: 100,
	})
	require.Equal(t, int64(9), s.items.Peek()[0].DownloadSpeed)
}

func TestStartMarksForgottenGidsError(t *testing.T) {
	require.NoError(t, db.OpenMemory(t.Context()))
	t.Cleanup(func() { db.Close() })
	mustUpsert(t, db.DB(), api.Download{GID: "a", Status: "active"})
	mustUpsert(t, db.DB(), api.Download{GID: "w", Status: "waiting"})

	s := NewService(settings.Settings{})
	fake := &fakeDownloader{
		known: map[string]api.Download{
			"a": {GID: "a", Status: "active"},
		}}
	s.downloader = fake
	require.NoError(t, s.Start(t.Context()))
	defer s.Close()

	var forgotten api.Download
	for _, d := range s.items.Peek() {
		if d.GID == "w" {
			forgotten = d
		}
	}
	require.Equal(t, "error", forgotten.Status)
	require.Equal(t, "download not found in aria2", forgotten.ErrorMessage)
	dbRow, err := getDownload(t.Context(), db.DB(), "w")
	require.NoError(t, err)
	require.Equal(t, "error", dbRow.Status, "marking persists")
	require.Equal(t, []string{"a"}, fake.listened, "only known gids re-listen")
}

func TestListenerErrorDedupes(t *testing.T) {
	s := NewService(settings.Settings{})
	var got []string
	s.OnError(func(err error) { got = append(got, err.Error()) })
	now := time.Now()

	boom := fmt.Errorf("websocket drop")
	s.listenerError("a", boom)
	s.listenerError("a", boom)
	require.Len(t, got, 1, "repeat within window stays silent")
	s.listenerError("b", boom)
	require.Len(t, got, 2, "other gid is a different storm")

	now = now.Add(31 * time.Second)
	s.listenerError("a", boom)
	require.Len(t, got, 3, "window expiry re-arms")
}
