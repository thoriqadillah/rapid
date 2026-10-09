package download

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"rapid/db"
	"rapid/lib/helpers/bools"
	"rapid/lib/helpers/iter"
	"rapid/services/download/api"
)

// Service is the application-facing download list for the widget layer.
//
// Phase 2 scope is read-only: it owns the newest-first download slice plus the
// active category/search filter and exposes them as plain Go values (no Qt
// model/view plumbing). Full action wiring (pause/resume/stop/delete/download/
// resolve/pick-folder) is intentionally absent; each will land as a method here
// marked // TODO(full-wire).
//
// db.DB() is a process singleton, so the service reads it directly instead of
// taking a DB handle.
type Service struct {
	mu       sync.RWMutex
	items    []api.Download
	category string
	search   string
	changed  []func()
}

// NewService builds an empty service backed by the process DB.
func NewService() *Service {
	return &Service{}
}

// All returns every download newest-first.
func (s *Service) All() []api.Download {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneDownloads(s.items)
}

// SetItems replaces the in-memory list (already ordered newest-first) and
// notifies subscribers.
func (s *Service) SetItems(items []api.Download) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.items = cloneDownloads(items)
	s.mu.Unlock()
	s.notify()
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

// ComputedItems returns the downloads matching the current category + search,
// preserving the newest-first order.
func (s *Service) ComputedItems() []api.Download {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return iter.Filter(s.items, func(d api.Download) bool {
		return s.matches(d, s.category, s.search)
	})
}

// SetCategory applies the sidebar filter ("" or "all" means all) and notifies.
func (s *Service) SetCategory(category string) {
	// sidebar "All downloads" uses destination "all"; normalize it
	// here so every caller routes through the same all-means-empty rule.
	category = bools.Ternary(category != "all", category, "")
	s.mu.Lock()
	if s.category == category {
		s.mu.Unlock()
		return
	}
	s.category = category
	s.mu.Unlock()
	s.notify()
}

// Category returns the active category filter.
func (s *Service) Category() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.category
}

// SetSearch applies a lowercased, trimmed search filter and notifies.
func (s *Service) SetSearch(query string) {
	query = strings.ToLower(strings.TrimSpace(query))
	s.mu.Lock()
	if s.search == query {
		s.mu.Unlock()
		return
	}
	s.search = query
	s.mu.Unlock()
	s.notify()
}

// Search returns the active (normalized) search filter.
func (s *Service) Search() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.search
}

// Counts aggregates the sidebar badges: "all" plus one entry per non-empty
// category.
func (s *Service) Counts() map[string]int {
	if s == nil {
		return map[string]int{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := map[string]int{"all": len(s.items)}
	return iter.Reduce(
		s.items,
		func(d api.Download, memo map[string]int) map[string]int {
			if category := string(d.Category); category != "" {
				memo[category]++
			}
			return memo
		},
		counts,
	)
}

// OnChanged subscribes to list/filter changes. Callbacks run on the caller's
// (GUI) thread; notify is only ever invoked from the thread that called
// SetCategory/SetSearch/Refresh.
//
// The returned function unsubscribes. Widget views MUST call it when they are
// destroyed: this service is a process singleton, so a page that subscribes on
// construction and is later recreated (e.g. navigation Replace) would otherwise
// retain every previous page's closure — and its whole widget tree — forever.
func (s *Service) OnChanged(fn func()) func() {
	if s == nil || fn == nil {
		return func() {}
	}
	s.mu.Lock()
	index := len(s.changed)
	s.changed = append(s.changed, fn)
	s.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			s.mu.Lock()
			if index < len(s.changed) {
				// nil the slot instead of reslicing: other subscribers keep
				// their captured indexes valid, and notify skips nils.
				s.changed[index] = nil
			}
			s.mu.Unlock()
		})
	}
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
	s.mu.Lock()
	if downloadsEqual(s.items, items) {
		s.mu.Unlock()
		return nil
	}
	s.items = items
	s.mu.Unlock()
	s.notify()
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

func (s *Service) notify() {
	s.mu.RLock()
	callbacks := make([]func(), 0, len(s.changed))
	for _, fn := range s.changed {
		if fn != nil {
			callbacks = append(callbacks, fn)
		}
	}
	s.mu.RUnlock()
	iter.ForEach(callbacks, func(fn func()) {
		fn()
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
