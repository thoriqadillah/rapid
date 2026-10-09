package errors

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConstructors(t *testing.T) {
	require.Equal(t, "x", NewAria2Error("x").Error())
	require.Equal(t, "n=1", Aria2Errorf("n=%d", 1).Error())
	require.Equal(t, "x", NewPluginError("x").Error())
	require.Equal(t, "t=x", PluginErrorf("t=%v", "x").Error())
	require.Equal(t, "x", NewTransportError("x").Error())
	require.Equal(t, "t=x", TransportErrorf("t=%v", "x").Error())
	require.Equal(t, "x", NewJsonRpcError(-32600, "x").Error())
	require.Equal(t, -32600, NewJsonRpcError(-32600, "x").Code)
	require.Equal(t, "x", NewBrowserRequestError("x").Error())
}
