package reactive

import (
	"reflect"
	"sync"
)

var (
	mu       sync.Mutex
	current  *observer // effect/computed currently collecting dependencies
	batching int
	pending  = map[*observer]struct{}{}
	depth    int
	epoch    uint64 // current top-level update; observers run at most once per epoch
)

// maxDepth bounds effect ping-pong: an effect that always dirties its own
// dependencies panics instead of hanging the GUI thread.
const maxDepth = 100

type node struct {
	subs map[*observer]struct{}
}

type observer struct {
	run      func()
	deps     map[*node]struct{}
	disposed bool
	firedAt  uint64 // epoch of last run; dedupes fan-out (one source, many memos)
}

// link subscribes o to n. Callers must hold mu.
func link(n *node, o *observer) {
	if _, ok := n.subs[o]; ok {
		return
	}
	if n.subs == nil {
		n.subs = map[*observer]struct{}{}
	}
	if o.deps == nil {
		o.deps = map[*node]struct{}{}
	}
	n.subs[o] = struct{}{}
	o.deps[n] = struct{}{}
}

// queueOrCollect registers n's observers for re-run: into the batch set when
// batching, otherwise returned for immediate firing. Callers must hold mu.
func queueOrCollect(n *node) []*observer {
	if batching > 0 {
		for o := range n.subs {
			pending[o] = struct{}{}
		}
		return nil
	}
	out := make([]*observer, 0, len(n.subs))
	for o := range n.subs {
		out = append(out, o)
	}
	return out
}

func fire(obs []*observer) {
	for _, o := range obs {
		fireOne(o)
	}
}

func fireOne(o *observer) {
	mu.Lock()
	if o.disposed || o.firedAt == epoch {
		mu.Unlock()
		return
	}
	o.firedAt = epoch
	if depth >= maxDepth {
		mu.Unlock()
		panic("reactive: possible infinite update loop")
	}
	depth++
	run := o.run
	mu.Unlock()

	defer func() {
		mu.Lock()
		depth--
		mu.Unlock()
	}()
	run()
}

// Signal is a read/write value cell. Same-value writes (per equal) are
// no-ops and notify nobody.
type Signal[T any] struct {
	node  node
	value T
	equal func(a, b T) bool
}

// NewSignal builds a cell with DeepEqual deduplication. Pass equal to
// override it (e.g. func(a, b T) bool { return false } keeps the old
// always-notify behavior).
func NewSignal[T any](initial T, equal ...func(a, b T) bool) *Signal[T] {
	eq := func(a, b T) bool {
		return reflect.DeepEqual(a, b)
	}

	if len(equal) > 0 && equal[0] != nil {
		eq = equal[0]
	}

	return &Signal[T]{
		value: initial,
		equal: eq,
	}
}

// Get reads the value, subscribing the enclosing effect/computed when inside one.
func (s *Signal[T]) Get() T {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		link(&s.node, current)
	}
	return s.value
}

// Peek reads without subscribing. Use it for one-off reads (logging,
// All(), accessors) so stray dependencies never form.
func (s *Signal[T]) Peek() T {
	mu.Lock()
	defer mu.Unlock()
	return s.value
}

// Set replaces the value and re-runs subscribers, unless equal.
func (s *Signal[T]) Set(v T) {
	mu.Lock()
	if s.equal(s.value, v) {
		mu.Unlock()
		return
	}
	s.value = v
	obs := queueOrCollect(&s.node)
	if obs != nil {
		epoch++ // new top-level update; nested fires inherit it
	}
	mu.Unlock()
	fire(obs)
}

// Effect runs fn immediately, then again whenever a signal read inside fn
// changes. It returns a disposer; views must call it on destroy.
func Effect(fn func()) (dispose func()) {
	o := &observer{}
	o.run = func() { runEffect(o, fn) }
	o.run()
	var once sync.Once
	return func() {
		once.Do(func() {
			mu.Lock()
			defer mu.Unlock()
			o.disposed = true
			for n := range o.deps {
				delete(n.subs, o)
			}
			o.deps = nil
		})
	}
}

func runEffect(o *observer, fn func()) {
	mu.Lock()
	if o.disposed {
		mu.Unlock()
		return
	}
	for n := range o.deps {
		delete(n.subs, o)
	}
	o.deps = nil
	prev := current
	current = o
	mu.Unlock()

	// Restore on panic too: a rendering bug must not permanently misattribute
	// every later dependency in the process graph.
	defer func() {
		mu.Lock()
		current = prev
		mu.Unlock()
	}()
	fn()
}

// Computed is a lazily recomputed memo. Dependencies invalidate it eagerly,
// the value itself recomputes on the next Get.
type Computed[T any] struct {
	node     node
	compute  func() T
	value    T
	dirty    bool
	inner    *observer
	disposed bool
}

// NewComputed builds a memo over compute. Computed values are cached per
// dependency state: repeated Gets without an intervening invalidation run
// compute exactly once.
func NewComputed[T any](compute func() T) *Computed[T] {
	c := &Computed[T]{
		compute: compute,
		dirty:   true,
	}

	c.inner = &observer{}
	c.inner.run = func() {
		mu.Lock()
		if c.disposed {
			mu.Unlock()
			return
		}
		c.dirty = true
		obs := queueOrCollect(&c.node)
		mu.Unlock()
		fire(obs)
	}
	c.refresh()
	return c
}

func (c *Computed[T]) refresh() {
	mu.Lock()
	if c.disposed {
		mu.Unlock()
		return
	}
	for n := range c.inner.deps {
		delete(n.subs, c.inner)
	}
	c.inner.deps = nil
	prev := current
	current = c.inner
	compute := c.compute
	mu.Unlock()

	// See runEffect: restore the tracker even when compute panics.
	defer func() {
		mu.Lock()
		current = prev
		mu.Unlock()
	}()

	v := compute()

	mu.Lock()
	c.value = v
	c.dirty = false
	mu.Unlock()
}

// Get reads the memo, subscribing the enclosing effect when inside one.
func (c *Computed[T]) Get() T {
	mu.Lock()
	if current != nil {
		link(&c.node, current)
	}
	dirty := c.dirty
	mu.Unlock()
	if dirty {
		c.refresh()
	}
	mu.Lock()
	defer mu.Unlock()
	return c.value
}

// Peek reads without subscribing.
func (c *Computed[T]) Peek() T {
	mu.Lock()
	dirty := c.dirty
	mu.Unlock()
	if dirty {
		c.refresh()
	}
	mu.Lock()
	defer mu.Unlock()
	return c.value
}

// Dispose unlinks the memo. Dispose subscribing effects first.
func (c *Computed[T]) Dispose() {
	mu.Lock()
	defer mu.Unlock()
	if c.disposed {
		return
	}
	c.disposed = true
	c.inner.disposed = true
	for n := range c.inner.deps {
		delete(n.subs, c.inner)
	}
	c.inner.deps = nil
}

// WritableComputed is a memo with a write-through setter: Get derives the
// value like Computed, Set runs set (which writes the underlying signals)
// inside one Batch so subscribers re-run once. The source signals stay the
// single source of truth; the memo itself is never assigned.
//
// This is the v-model piece: a widget binds Get for display and calls Set on
// user input, e.g. a search field over a normalized query:
//
//	model := reactive.NewWritableComputed(
//		func() string { return query.Get() },
//		func(v string) { query.Set(strings.ToLower(strings.TrimSpace(v))) },
//	)
type WritableComputed[T any] struct {
	*Computed[T]
	set func(T)
}

// NewWritableComputed builds a get/set memo. A nil set is a no-op write.
func NewWritableComputed[T any](get func() T, set func(T)) *WritableComputed[T] {
	return &WritableComputed[T]{
		Computed: NewComputed(get),
		set:      set,
	}
}

// Set writes through to the source signals. Same-value writes are no-ops:
// when set leaves the sources unchanged, nobody is notified.
func (c *WritableComputed[T]) Set(v T) {
	set := c.set
	Batch(func() { set(v) })
}

// Get reads the memo. (Explicit forwarder: promoted methods would dereference
// a nil outer pointer before reaching Computed's own nil guard.)
func (c *WritableComputed[T]) Get() T {
	return c.Computed.Get()
}

// Peek reads without subscribing.
func (c *WritableComputed[T]) Peek() T {
	return c.Computed.Peek()
}

// Dispose unlinks the memo.
func (c *WritableComputed[T]) Dispose() {
	c.Computed.Dispose()
}

// Model is either end of a binding: anything readable and writable. Both
// Signal and WritableComputed satisfy it.
type Model[T any] interface {
	Get() T
	Set(T)
}

// Bind returns a new signal two-way synced with source, like Vue's
// defineModel for a parent's v-model: writing either side appears in the
// other. That covers signal-to-signal as well as signal-to-writable-model
// pairs; the child is always a plain Signal.
//
// Teardown is symmetric: the disposer splits both directions, so after it
// neither side sees the other.
// A nil source yields a detached zero signal and a no-op disposer.
//
// Termination relies on value equality, not an in-flight flag: a write is
// forwarded only when the other end actually differs (DeepEqual), so a
// same-value echo — including from a source with an always-false equal —
// is a no-op instead of ping-ponging. A differing echo, e.g. a normalizing
// setter turning "Dune" into "dune", propagates exactly one extra hop and
// settles. Only a setter that transforms on every trip keeps the loop alive,
// and that hits the loop guard instead of hanging.
// The child adopts the source's current value at bind time.
func Bind[T any](source Model[T]) (*Signal[T], func()) {
	if source == nil {
		return NewSignal(*new(T)), func() {}
	}
	child := NewSignal(source.Get())
	return child, linkModels(child, source)
}

// linkModels syncs a and b bidirectionally; see Bind. Both ends must be
// non-nil and already agree; Bind establishes that.
func linkModels[T any](a, b Model[T]) (dispose func()) {
	oneWay := func(from, to Model[T]) func() {
		return Effect(func() {
			// Guard read is untracked: each direction subscribes to its
			// source only, or every write would re-run both effects.
			if v := from.Get(); !reflect.DeepEqual(Untrack(to.Get), v) {
				to.Set(v)
			}
		})
	}
	unA := oneWay(a, b)
	unB := oneWay(b, a)
	var once sync.Once
	return func() {
		once.Do(func() {
			unA()
			unB()
		})
	}
}

// Batch groups writes: subscribers re-run once when the outermost Batch
// returns instead of once per Set.
func Batch(fn func()) {
	mu.Lock()
	batching++
	mu.Unlock()

	// Flush on panic too: queued observers must still invalidate, or memos
	// keep serving stale values after the programmer fixes the bug.
	defer func() {
		mu.Lock()
		batching--
		var obs []*observer
		if batching == 0 {
			for o := range pending {
				obs = append(obs, o)
			}
			pending = map[*observer]struct{}{}
			if len(obs) > 0 {
				epoch++ // one flush, one epoch: each observer runs once
			}
		}
		mu.Unlock()
		fire(obs)
	}()
	fn()
}

// Untrack runs fn without collecting dependencies.
func Untrack[T any](fn func() T) T {
	mu.Lock()
	prev := current
	current = nil
	mu.Unlock()
	defer func() {
		mu.Lock()
		current = prev
		mu.Unlock()
	}()
	return fn()
}
