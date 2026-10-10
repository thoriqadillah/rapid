package download

import (
	"context"
	"path/filepath"
	"strings"

	"rapid/db"
	"rapid/lib/helpers/bools"
	"rapid/lib/helpers/iter"
	"rapid/lib/reactive"
	"rapid/services/download/api"
)

// Service is the application-facing download list for the widget layer.
//
// It owns the newest-first download slice plus the active category/search
// filter as reactive signals. Items/Counts are memos over them; views either
// subscribe with OnChanged (one Effect per subscriber) or bind the memos
// directly into widgets (FilteredItems/Counts), and dispose on
// destroy. All methods must run on the GUI thread; effects fire synchronously
// on the writer's goroutine (see lib/reactive).
//
// db.DB() is a process singleton, so the service reads it directly instead of
// taking a DB handle.
type Service struct {
	items         *reactive.Signal[[]api.Download]
	Category      *reactive.Signal[string]
	Search        *reactive.Signal[string]
	Items         *reactive.Computed[[]api.Download]
	Counts        *reactive.Computed[map[string]int]
	FilterChanged *reactive.Computed[bool]
}

// NewService builds an empty service backed by the process DB.
func NewService() *Service {
	s := &Service{
		items:    reactive.NewSignal([]api.Download(nil)),
		Category: reactive.NewSignal(""),
		Search:   reactive.NewSignal(""),
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
		s.Items.Get()
		s.Counts.Get()
		return true
	})
	return s
}

// SetItems replaces the in-memory list (already ordered newest-first) and
// notifies subscribers.
func (s *Service) SetItems(items []api.Download) {
	s.items.Set(cloneDownloads(items))
}

func (s *Service) matches(d api.Download, category, search string) bool {
	if category != "" && category != "all" && string(d.Category) != category {
		return false
	}
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

// SetCategory applies the sidebar filter ("" or "all" means all) and notifies.
func (s *Service) SetCategory(category string) {
	category = bools.Ternary(category != "all", category, "")
	s.Category.Set(category)
}

// SetSearch applies a lowercased, trimmed search filter and notifies.
func (s *Service) SetSearch(query string) {
	query = strings.ToLower(strings.TrimSpace(query))
	s.Search.Set(query)
}

// OnChanged subscribes to list/filter changes. The callback runs on the
// caller's (GUI) thread; SetCategory/SetSearch/SetItems/Refresh must all run
// there.
//
// The returned function unsubscribes (it disposes the underlying Effect).
// Widget views MUST call it when they are destroyed: the graph retains every
// live effect — and its whole widget closure — until disposed.
func (s *Service) OnChanged(fn func()) func() {
	if s == nil || fn == nil {
		return func() {}
	}
	// Read through both memos so the effect tracks every source
	// (items via filtered+counts, category/search via filtered). The first
	// run only collects dependencies; callbacks fire on later re-runs.
	first := true
	return reactive.Effect(func() {
		s.Items.Get()
		s.Counts.Get()
		if first {
			first = false
			return
		}
		fn()
	})
}

// Refresh reloads every download from the process DB, newest-first. It notifies
// subscribers only when the data actually changed: the widget layer polls this
// on a timer, and notifying unconditionally forced a full re-render (and its
// native Qt allocations) every tick even when nothing had moved.
func (s *Service) Refresh(ctx context.Context) error {
	items, err := getAllDownloads(ctx, db.DB())
	if err != nil {
		return err
	}
	if downloadsEqual(s.items.Peek(), items) {
		return nil
	}
	s.items.Set(items)
	return nil
}

func downloadsEqual(a, b []api.Download) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].Equal(b[i]) {
			return false
		}
	}
	return true
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
