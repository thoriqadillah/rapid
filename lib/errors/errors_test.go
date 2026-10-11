package errors

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstructors(t *testing.T) {
	require.Equal(t, "x", NewAria2Error("x").Error())
	require.Equal(t, "n=1", Aria2Errorf("n=%d", 1).Error())
	require.Equal(t, "x", NewPluginError("x").Error())
	require.Equal(t, "t=x", PluginErrorf("t=%v", "x").Error())
	require.Equal(t, "x", NewTransportError("x").Error())
	require.Equal(t, "x", NewJsonRpcError(-32600, "x").Error())
	require.Equal(t, -32600, NewJsonRpcError(-32600, "x").Code)
	require.Equal(t, "x", NewBrowserRequestError("x").Error())
}

// Wrapped causes stay reachable via errors.Is through every typed error, and
// Error() shows both message and cause.
func TestWrapErrorsExposeCause(t *testing.T) {
	cases := []struct {
		name  string
		err   error
		cause error
		text  string
	}{
		{"aria2", NewAria2Error("cannot connect to aria2", context.Canceled),
			context.Canceled, "cannot connect to aria2: context canceled"},
		{"plugin", NewPluginError("resolver failed", context.DeadlineExceeded),
			context.DeadlineExceeded, "resolver failed: context deadline exceeded"},
		{"transport", NewTransportError("dial failed", context.Canceled),
			context.Canceled, "dial failed: context canceled"},
	}
	for _, c := range cases {
		require.True(t, stderrors.Is(c.err, c.cause), c.name)
		require.EqualError(t, c.err, c.text, c.name)
	}
}

func TestIsAria2NotFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"invalid gid message", NewAria2Error("aria2 error code 1: Invalid GID dead-gid"), true},
		{"not found message", NewAria2Error("aria2 error code 1: GID#2089b05ecca3d829 is not found"), true},
		{"other aria2 error", NewAria2Error("aria2 error code 1: hey"), false},
		{"unrelated error", fmt.Errorf("dial tcp: refused"), false},
		{"wrapped in chain", fmt.Errorf("poll: %w", NewAria2Error("GID#abc is not found")), true},
		{"wrapped cause, no aria2 type", NewTransportError("dial", context.Canceled), false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, IsAria2NotFound(c.err), c.name)
	}
}
