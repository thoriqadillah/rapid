package errors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConstructors(t *testing.T) {
	assert.Equal(t, "x", NewAria2Error("x").Error())
	assert.Equal(t, "n=1", Aria2Errorf("n=%d", 1).Error())
	assert.Equal(t, "x", NewPluginError("x").Error())
	assert.Equal(t, "t=x", PluginErrorf("t=%v", "x").Error())
	assert.Equal(t, "x", NewTransportError("x").Error())
	assert.Equal(t, "t=x", TransportErrorf("t=%v", "x").Error())
	assert.Equal(t, "x", NewJsonRpcError(-32600, "x").Error())
	assert.Equal(t, -32600, NewJsonRpcError(-32600, "x").Code)
	assert.Equal(t, "x", NewBrowserRequestError("x").Error())
}
