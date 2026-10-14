// Package signal provides Qt-signal-like typed events: Connect registers a
// slot and returns its disconnect func, Emit fans out to all connected
// slots. Unlike reactive signals, an event has no current value, no initial
// fire, and no dedup — every Emit reaches every slot exactly once, which is
// what one-shot occurrences (completed/failed) need.
package signal

// Event is a typed fan-out with connect/emit/disconnect vocabulary. The zero
// value is usable. Not safe for concurrent use; owners emit on one thread
// (the download service emits on the GUI thread).
type Event[T any] struct {
	subs []func(T)
}

// Connect registers a slot, returning its disconnect func (idempotent).
// A nil fn registers nothing.
func (e *Event[T]) Connect(fn func(T)) (disconnect func()) {
	if fn == nil {
		return func() {}
	}
	e.subs = append(e.subs, fn)
	i := len(e.subs) - 1
	return func() { e.subs[i] = nil }
}

// Emit calls every connected slot in order, skipping disconnected ones.
func (e *Event[T]) Emit(v T) {
	for _, fn := range e.subs {
		if fn != nil {
			fn(v)
		}
	}
}
