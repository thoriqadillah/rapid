package signal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventFansOutAndDisconnects(t *testing.T) {
	var e Event[string]
	var got []string
	e.Connect(func(v string) { got = append(got, "a:"+v) })
	drop := e.Connect(func(v string) { got = append(got, "b:"+v) })
	e.Connect(nil) // safe no-op

	e.Emit("x")
	require.Equal(t, []string{"a:x", "b:x"}, got)

	drop()
	drop() // idempotent
	e.Emit("y")
	require.Equal(t, []string{"a:x", "b:x", "a:y"}, got)
}

func TestEventZeroValueEmitsToNobody(t *testing.T) {
	var e Event[int]
	require.NotPanics(t, func() { e.Emit(1) })
}
