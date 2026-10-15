package download

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	stderr "errors"
	"rapid/db"
	"rapid/lib/errors"
	"rapid/lib/helpers/bools"
	"rapid/lib/helpers/iter"
	"rapid/lib/reactive"
	"rapid/lib/signal"
	"rapid/lib/worker"
	"rapid/services/download/api"
	"rapid/services/download/downloader"
	"rapid/services/download/rpc"
	"rapid/services/plugin"
	"rapid/services/settings"

	"github.com/mappu/miqt/qt6/mainthread"
	"github.com/uptrace/bun"
)

type Service struct {
	items         *reactive.Signal[[]api.Download]
	Category      *reactive.Signal[string]
	Search        *reactive.Signal[string]
	Items         *reactive.Computed[[]api.Download]
	Counts        *reactive.Computed[map[string]int]
	FilterChanged *reactive.Computed[bool]
	ActiveCount   *reactive.Computed[int]

	settings   settings.Settings
	downloader downloader.Downloader
	fallback   downloader.Resolver
	plugins    *plugin.Manager // nil until S8 constructs it here
	queue      *worker.Queue

	completed signal.Event[api.Download]
	error     signal.Event[error]
}

func NewService(settings settings.Settings) *Service {
	client := rpc.NewJSONRpc(settings.Aria2Host, settings.Aria2Port, settings.Aria2Token)
	dl := downloader.NewAria2Downloader(settings, client, true, nil /*globalNotify unused for now*/)
	s := &Service{
		items:      reactive.NewSignal([]api.Download(nil)),
		Category:   reactive.NewSignal(""),
		Search:     reactive.NewSignal(""),
		settings:   settings,
		downloader: dl,
		fallback:   dl, // aria2 doubles as the probe resolver (S9)
		plugins:    nil,
		queue:      worker.NewQueue(16),
	}
	s.Items = reactive.NewComputed(func() []api.Download {
		items, category, search := s.items.Get(), s.Category.Get(), s.Search.Get()
		return iter.Filter(items, func(d api.Download) bool {
			return s.matches(d, category, search)
		})
	})
	s.Counts = reactive.NewComputed(func() map[string]int {
		items := s.items.Get()
		counts := map[string]int{"all": len(items)}
		return iter.Reduce(
			items,
			func(d api.Download, memo map[string]int) map[string]int {
				if category := d.Category.String(); category != "" {
					memo[category]++
				}
				return memo
			},
			counts,
		)
	})
	s.FilterChanged = reactive.NewComputed(func() bool {
		s.Category.Get()
		s.Search.Get()
		return true
	})
	s.ActiveCount = reactive.NewComputed(func() int {
		return iter.Reduce(
			s.items.Get(),
			func(d api.Download, n int) int {
				if d.Status == "active" {
					return n + 1
				}
				return n
			},
			0,
		)
	})
	return s
}

// Start boots the action queue and the downloader, loads the DB rows, and
// re-listens on restored rows the daemon still knows. Called once at startup
// on the GUI thread; steady-state control ops go through the queue instead.
func (s *Service) Start(ctx context.Context) error {
	s.queue.Start(ctx)
	items, err := getAllDownloads(ctx, db.DB())
	if err != nil {
		return err
	}

	s.items.Set(items)
	if err := s.downloader.Start(ctx); err != nil {
		return err
	}
	for _, d := range items {
		if d.Status != "active" && d.Status != "waiting" {
			continue
		}
		if _, err := s.downloader.GetStatus(ctx, d.GID); err != nil {
			slog.Warn("download: daemon forgot gid", "gid", d.GID, "err", err)
			continue
		}
		s.downloader.Listen(d.GID, s.onListenerNotify, s.onListenerError)
	}
	return nil
}

// Close stops the action queue (pending tasks are abandoned) then the
// downloader (which saves the aria2 session). Runs at quit. Idempotent.
func (s *Service) Close() {
	s.queue.Close()
	if err := s.downloader.Stop(); err != nil {
		slog.Error("download: stop downloader", "err", err)
	}
}

// Pause suspends gid and stops listening for it; the row keeps its state
// with no speed line until resumed or removed.
func (s *Service) Pause(ctx context.Context, gid string) {
	s.queue.Go(func(ctx context.Context) {
		d, err := s.downloader.Pause(ctx, gid)
		if err != nil {
			mainthread.Start(func() { s.emitError("pause", gid, err) })
			return
		}
		s.downloader.Unlisten(gid)
		mainthread.Start(func() { s.upsertDownload(d) })
	})
}

// Resume restarts gid and listens for it again.
func (s *Service) Resume(ctx context.Context, gid string) {
	s.queue.Go(func(ctx context.Context) {
		d, err := s.downloader.Resume(ctx, gid)
		if err != nil {
			mainthread.Start(func() { s.emitError("resume", gid, err) })
			return
		}
		s.downloader.Listen(gid, s.onListenerNotify, s.onListenerError)
		mainthread.Start(func() { s.upsertDownload(d) })
	})
}

// Stop halts gid but retains its removed row until Delete purges it.
func (s *Service) Stop(ctx context.Context, gid string) {
	s.queue.Go(func(ctx context.Context) {
		d, err := s.downloader.Remove(ctx, gid)
		if err != nil {
			mainthread.Start(func() { s.emitError("stop", gid, err) })
			return
		}
		s.downloader.Unlisten(gid)
		mainthread.Start(func() { s.upsertDownload(d) })
	})
}

// Delete purges gid from the daemon and drops its row (and optionally its
// files). Files are snapshotted here on the GUI thread: the queued op can't
// read in-memory state later.
func (s *Service) Delete(ctx context.Context, gid string, fromDisk bool) {
	s.queue.Go(func(ctx context.Context) {
		d, ok := iter.Find(s.items.Peek(), func(d api.Download) bool { return d.GID == gid })
		if !ok {
			return
		}

		if _, err := s.downloader.Remove(ctx, gid); err != nil && !errors.IsAria2NotFound(err) {
			mainthread.Start(func() { s.emitError("delete", gid, err) })
			return
		}
		if err := s.downloader.Purge(ctx, gid); err != nil && !errors.IsAria2NotFound(err) {
			mainthread.Start(func() { s.emitError("delete", gid, err) })
			return
		}
		s.downloader.Unlisten(gid)
		if fromDisk {
			removeFiles(d.Files)
		}

		mainthread.Start(func() {
			if err := removeDownload(context.Background(), db.DB(), gid); err != nil {
				s.emitError("delete", gid, err)
				return
			}
			newItems := iter.Filter(s.items.Peek(), func(d api.Download) bool { return d.GID != gid })
			s.items.Set(newItems)
		})
	})
}

// emitError reports a failed control op with the download's display name.
// GUI thread only in production (called marshaled); plain Go otherwise, so
// tests call it directly.
func (s *Service) emitError(action, gid string, err error) {
	d, ok := iter.Find(s.items.Peek(), func(d api.Download) bool { return d.GID == gid })
	if ok {
		s.error.Emit(fmt.Errorf("Failed to %s %s: %w", action, d.Name(), err))
	}
}

func (s *Service) matches(d api.Download, category, search string) bool {
	category = bools.Ternary(category != "all", category, "")
	if category != "" && string(d.Category) != category {
		return false
	}

	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return true
	}

	if d.Resolved != nil {
		title := d.Resolved.Title
		if title == "" {
			title = d.Resolved.Filename
		}
		if strings.Contains(strings.ToLower(title), search) {
			return true
		}
	}

	if p := d.FirstPath(); p != "" {
		if strings.Contains(strings.ToLower(filepath.Base(p)), search) {
			return true
		}
	}
	return false
}

func (s *Service) onListenerNotify(d api.Download) {
	mainthread.Start(func() {
		s.upsertDownload(d)
	})
}

// upsertDownload folds a downloader notification into DB + memory + user
// events. GUI thread only: DB writes stay here, state has no lock.
func (s *Service) upsertDownload(d api.Download) {
	items := s.items.Peek()
	i := iter.FindIndex(items, func(it api.Download) bool { return it.GID == d.GID })
	var prev api.Download
	if i >= 0 {
		prev = items[i]
		// aria2 doesn't know these; keep the previous row's when fresh lacks them.
		d.Category = bools.Ternary(d.Category != "", d.Category, prev.Category)
		d.Resolved = bools.Ternary(d.Resolved != nil, d.Resolved, prev.Resolved)
	}

	var merged api.Download
	err := db.DB().RunInTx(context.Background(), nil, func(ctx context.Context, tx bun.Tx) error {
		var err error
		merged, err = upsertDownload(ctx, tx, d)
		if err != nil {
			return err
		}
		if d.Status == "active" && d.DownloadSpeed > 0 {
			return addSpeedSample(ctx, tx, d.GID, time.Now().UnixMilli(), d.DownloadSpeed)
		}
		return nil
	})

	if err != nil {
		slog.Error("download: persist notified row", "gid", d.GID, "err", err)
		s.emitError("download", d.GID, err)
		return
	}

	next := cloneItems(s.items.Peek())
	if i >= 0 {
		next[i] = merged
	} else {
		next = append([]api.Download{merged}, next...) // newest-first
	}
	s.items.Set(next)

	// Terminal transitions fire exactly once; a poll refresh can redeliver
	// the same terminal status.
	if !prev.IsFinished() && merged.IsFinished() {
		s.completed.Emit(merged)
	}

	if !prev.IsError() && merged.IsError() {
		s.emitError("download", d.GID, stderr.New(merged.ErrorMessage))
	}
}

func (s *Service) OnCompleted(fn func(api.Download)) (disconnect func()) {
	return s.completed.Connect(fn)
}

func (s *Service) OnError(fn func(error)) (disconnect func()) {
	return s.error.Connect(fn)
}

func (s *Service) onListenerError(err error) {
	mainthread.Start(func() {
		slog.Error("download service error", "task", "Download service", "err", err)
	})
}

// SpeedHistory returns the last 60 samples for gid in ascending ts order.
func (s *Service) SpeedHistory(ctx context.Context, gid string) ([]api.SpeedSample, error) {
	return getSpeedHistory(ctx, db.DB(), gid, 100)
}

// SpeedSamples is the sparkline-friendly view of SpeedHistory.
func (s *Service) SpeedSamples(ctx context.Context, gid string) []int64 {
	samples, err := s.SpeedHistory(ctx, gid)
	if err != nil {
		return nil
	}

	return iter.Map(samples, func(s api.SpeedSample) int64 {
		return s.Speed
	})
}

// TODO(full-wire): Download/Resolve/PickFolder. These will operate through
// the downloader and update the in-memory slice + store, then notify.
