// Package errors defines the typed backend errors. Each type has constructor
// functions — construct, don't struct-literal. Message text is matched by
// tests and the extension error contract, so keep it stable.
package errors

import (
	"fmt"
)

// Aria2Error is a daemon/RPC failure.
type Aria2Error struct {
	Msg string
}

func (e *Aria2Error) Error() string {
	return e.Msg
}

// NewAria2Error builds an Aria2Error.
func NewAria2Error(msg string) *Aria2Error {
	return &Aria2Error{Msg: msg}
}

// Aria2Errorf builds an Aria2Error with formatting.
func Aria2Errorf(format string, args ...any) *Aria2Error {
	return &Aria2Error{Msg: fmt.Sprintf(format, args...)}
}

// PluginError is raised when a resolver plugin cannot run or answers invalidly.
type PluginError struct {
	Msg string
}

func (e *PluginError) Error() string {
	return e.Msg
}

// NewPluginError builds a PluginError.
func NewPluginError(msg string) *PluginError {
	return &PluginError{Msg: msg}
}

// PluginErrorf builds a PluginError with formatting.
func PluginErrorf(format string, args ...any) *PluginError {
	return &PluginError{Msg: fmt.Sprintf(format, args...)}
}

// TransportError is raised when a plugin transport cannot send or receive.
type TransportError struct {
	Msg string
}

func (e *TransportError) Error() string {
	return e.Msg
}

// NewTransportError builds a TransportError.
func NewTransportError(msg string) *TransportError {
	return &TransportError{Msg: msg}
}

// TransportErrorf builds a TransportError with formatting.
func TransportErrorf(format string, args ...any) *TransportError {
	return &TransportError{Msg: fmt.Sprintf(format, args...)}
}

// JsonRpcError is raised when a plugin replies with an error or garbage.
type JsonRpcError struct {
	Code    int
	Message string
}

func (e *JsonRpcError) Error() string {
	return e.Message
}

// NewJsonRpcError builds a JsonRpcError.
func NewJsonRpcError(code int, message string) *JsonRpcError {
	return &JsonRpcError{Code: code, Message: message}
}

// BrowserRequestError is raised for unsafe or invalid extension requests.
type BrowserRequestError struct {
	Msg string
}

func (e *BrowserRequestError) Error() string {
	return e.Msg
}

// NewBrowserRequestError builds a BrowserRequestError.
func NewBrowserRequestError(msg string) *BrowserRequestError {
	return &BrowserRequestError{Msg: msg}
}
