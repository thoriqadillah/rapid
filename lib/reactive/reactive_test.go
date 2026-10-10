package reactive

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignalGetSetPeek(t *testing.T) {
	s := NewSignal(1)
	require.Equal(t, 1, s.Get())
	s.Set(2)
	require.Equal(t, 2, s.Get())
	require.Equal(t, 2, s.Peek())
}

func TestSignalSameValueIsNoOp(t *testing.T) {
	s := NewSignal("a")
	var calls int
	dispose := Effect(func() {
		s.Get()
		calls++
	})
	defer dispose()
	require.Equal(t, 1, calls)

	s.Set("a")
	require.Equal(t, 1, calls, "same-value write must not re-run")

	s.Set("b")
	require.Equal(t, 2, calls)
}

func TestEffectTracksOnlyReadSignals(t *testing.T) {
	a := NewSignal(1)
	b := NewSignal(10)
	var got int
	dispose := Effect(func() {
		got = a.Get() + Untrack(b.Get)
	})
	defer dispose()
	require.Equal(t, 11, got)

	b.Set(20)
	require.Equal(t, 11, got, "untracked write must not re-run")

	a.Set(2)
	require.Equal(t, 22, got)
}

func TestEffectDispose(t *testing.T) {
	s := NewSignal(0)
	var calls int
	dispose := Effect(func() {
		_ = s.Get()
		calls++
	})
	require.Equal(t, 1, calls)

	dispose()
	dispose() // safe twice
	s.Set(1)
	require.Equal(t, 1, calls, "disposed effect must not fire")
}

func TestComputedMemoizes(t *testing.T) {
	s := NewSignal(2)
	var runs int
	double := NewComputed(func() int {
		runs++
		return s.Get() * 2
	})
	defer double.Dispose()

	require.Equal(t, 4, double.Get())
	require.Equal(t, 4, double.Get())
	require.Equal(t, 1, runs, "repeat Gets must not recompute")

	s.Set(3)
	require.Equal(t, 6, double.Get())
	require.Equal(t, 2, runs)
}

func TestComputedChainsAndEffects(t *testing.T) {
	s := NewSignal(1)
	double := NewComputed(func() int { return s.Get() * 2 })
	defer double.Dispose()
	quad := NewComputed(func() int { return double.Get() * 2 })
	defer quad.Dispose()

	var got int
	dispose := Effect(func() { got = quad.Get() })
	defer dispose()
	require.Equal(t, 4, got)

	s.Set(2)
	require.Equal(t, 8, got)
}

func TestBatchCoalesces(t *testing.T) {
	a := NewSignal(0)
	b := NewSignal(0)
	var calls int
	dispose := Effect(func() {
		a.Get()
		b.Get()
		calls++
	})
	defer dispose()
	require.Equal(t, 1, calls)

	Batch(func() {
		a.Set(1)
		b.Set(2)
		callsCheck := calls
		require.Equal(t, 1, callsCheck, "no re-run inside batch")
	})
	require.Equal(t, 2, calls, "one re-run per batch")
}

func TestFanOutFiresOnce(t *testing.T) {
	s := NewSignal(1)
	a := NewComputed(func() int { return s.Get() + 1 })
	defer a.Dispose()
	b := NewComputed(func() int { return s.Get() + 2 })
	defer b.Dispose()

	var calls int
	dispose := Effect(func() {
		a.Get()
		b.Get()
		calls++
	})
	defer dispose()
	require.Equal(t, 1, calls)

	// One source write fans out through two memos; the effect still runs once.
	s.Set(2)
	require.Equal(t, 2, calls)
	require.Equal(t, 3, a.Get())
	require.Equal(t, 4, b.Get())
}

func TestLoopGuardPanicsAndRecovers(t *testing.T) {
	s := NewSignal(0)
	require.Panics(t, func() {
		Effect(func() {
			s.Set(s.Get() + 1)
		})
	})

	// Graph stays usable after the panic: depth counter was restored.
	s2 := NewSignal(0)
	var calls int
	dispose := Effect(func() {
		_ = s2.Get()
		calls++
	})
	defer dispose()
	s2.Set(1)
	require.Equal(t, 2, calls)
}

func TestWritableComputedReadsThrough(t *testing.T) {
	query := NewSignal("")
	model := NewWritableComputed(
		func() string { return query.Get() },
		func(v string) { query.Set(v) },
	)
	defer model.Dispose()

	require.Equal(t, "", model.Get())
	query.Set("dune")
	require.Equal(t, "dune", model.Get())
}

func TestWritableComputedWritesThroughOnce(t *testing.T) {
	a := NewSignal("")
	b := NewSignal("")
	model := NewWritableComputed(
		func() string { return a.Get() + b.Get() },
		func(v string) {
			a.Set(v)
			b.Set(v)
		},
	)
	defer model.Dispose()

	var calls int
	dispose := Effect(func() {
		model.Get()
		calls++
	})
	defer dispose()
	require.Equal(t, 1, calls)

	// One setter writing two signals still notifies once (batched).
	model.Set("x")
	require.Equal(t, "xx", model.Get())
	require.Equal(t, 2, calls)

	// Same-value write through a normalizing setter is a no-op.
	norm := NewWritableComputed(
		func() string { return a.Get() },
		func(v string) {
			if v == "bad" {
				return // rejected: sources untouched
			}
			a.Set(v)
		},
	)
	defer norm.Dispose()
	norm.Set("bad")
	require.Equal(t, 2, calls, "rejected write must not notify")
}

func TestBindSyncsBothWays(t *testing.T) {
	parent := NewSignal("")
	child, unbind := Bind(parent)
	defer unbind()
	require.Equal(t, "", child.Get(), "child must adopt source at bind time")

	parent.Set("x")
	require.Equal(t, "x", child.Get(), "parent -> child did not sync")

	child.Set("y")
	require.Equal(t, "y", parent.Get(), "child -> parent did not sync")
}

func TestBindTerminatesWithoutEquality(t *testing.T) {
	// An always-false equal would ping-pong forever on value equality alone;
	// the bind flag must settle the echo after one round trip.
	never := func(a, b string) bool { return false }
	parent := NewSignal("", never)
	var runs int
	done := Effect(func() {
		parent.Get()
		runs++
	})
	defer done()
	child, unbind := Bind(parent)
	defer unbind()

	base := runs
	parent.Set("x")
	require.Equal(t, "x", child.Get())
	require.LessOrEqual(t, runs-base, 2, "echo did not settle promptly")

	child.Set("y")
	require.Equal(t, "y", parent.Get())
}

func TestBindSignalToModel(t *testing.T) {
	raw := NewSignal("")
	upper := NewWritableComputed(
		func() string { return raw.Get() },
		func(v string) { raw.Set(v) },
	)
	defer upper.Dispose()

	child, unbind := Bind(upper)
	defer unbind()
	require.Equal(t, "", child.Get(), "child must adopt the model")

	child.Set("hi")
	require.Equal(t, "hi", upper.Get())
	require.Equal(t, "hi", raw.Peek())

	raw.Set("yo")
	require.Equal(t, "yo", child.Get())
}

func TestBindDispose(t *testing.T) {
	parent := NewSignal(0)
	child, unbind := Bind(parent)

	unbind()
	unbind() // safe twice
	parent.Set(1)
	require.Equal(t, 0, child.Get(), "disposed bind still syncs parent -> child")
	child.Set(2)
	require.Equal(t, 1, parent.Get(), "disposed bind still syncs child -> parent")
}
