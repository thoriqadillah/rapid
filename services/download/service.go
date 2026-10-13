package download

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	"rapid/db"
	"rapid/lib/helpers/bools"
	"rapid/lib/helpers/iter"
	"rapid/lib/reactive"
	"rapid/lib/worker"
	"rapid/services/download/api"
	"rapid/services/download/downloader"
	"rapid/services/download/rpc"
	"rapid/services/plugin"
	"rapid/services/settings"

	"github.com/mappu/miqt/qt6/mainthread"
)

type Service struct {
	items         *reactive.Signal[[]api.Download]
	Category      *reactive.Signal[string]
	Search        *reactive.Signal[string]
	Items         *reactive.Computed[[]api.Download]
	Counts        *reactive.Computed[map[string]int]
	FilterChanged *reactive.Computed[bool]

	settings   settings.Settings
	downloader downloader.Downloader
	fallback   downloader.Resolver
	plugins    *plugin.Manager // nil until S8 constructs it here
	queue      *worker.Queue
}

func NewService(settings settings.Settings) *Service {
	client := rpc.NewJSONRpc(settings.Aria2Host, settings.Aria2Port, settings.Aria2Token)
	dl := downloader.NewAria2Downloader(settings, client, true /*manageDaemon*/, nil /*globalNotify unused for now*/)
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
		i := iter.FindIndex(s.items.Peek(), func(it api.Download) bool { return it.GID == d.GID })
		if i < 0 {
			return
		}

		newItems := cloneDownloads(s.items.Peek())
		newItems[i] = d
		s.items.Set(newItems)
	})
}

// onListenerError funnels listener errors to notifyError on the GUI thread.
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

func cloneDownloads(items []api.Download) []api.Download {
	if items == nil {
		return nil
	}
	out := make([]api.Download, len(items))
	copy(out, items)
	return out
}

// TODO(full-wire): Pause/Resume/Stop/Delete/Download/Resolve/PickFolder/
// ActiveCount/ActiveDownloads. These will operate through the downloader and
// update the in-memory slice + store, then notify.
