package plugin

// PluginError is a plugin failure with the offending plugin attached. It wraps
// the cause (transport/JSON-RPC/decode failures, including lib/errors
// types), so errors.Is/As keep working through it.
type PluginError struct {
	PluginID string
	Err      error
}

func NewPluginError(pluginID string, err error) *PluginError {
	return &PluginError{
		PluginID: pluginID,
		Err:      err,
	}
}

func (e *PluginError) Error() string {
	if e.Err == nil {
		return e.PluginID
	}
	return e.PluginID + ": " + e.Err.Error()
}

func (e *PluginError) Unwrap() error {
	return e.Err
}
