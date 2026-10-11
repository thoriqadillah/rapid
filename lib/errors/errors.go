// Package errors defines the typed backend errors. Each type has constructor
// functions — construct, don't struct-literal. Message text is matched by
// tests and the extension error contract, so keep it stable.
package errors

import (
	stdErrors "errors"
	"fmt"
	"strings"
)

// Aria2Error is a daemon/RPC failure.
type Aria2Error struct {
	Msg string
	Err error
}

func (e *Aria2Error) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	return e.Msg + ": " + e.Err.Error()
}

// Unwrap exposes the cause so errors.Is/As (e.g. context.Canceled) work.
func (e *Aria2Error) Unwrap() error {
	return e.Err
}

// NewAria2Error builds an Aria2Error.
func NewAria2Error(msg string, errs ...error) *Aria2Error {
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}

	return &Aria2Error{Msg: msg, Err: err}
}

// Aria2Errorf builds an Aria2Error with formatting.
func Aria2Errorf(format string, args ...any) *Aria2Error {
	return &Aria2Error{Msg: fmt.Sprintf(format, args...)}
}

// PluginError is raised when a resolver plugin cannot run or answers invalidly.
type PluginError struct {
	Msg string
	Err error
}

func (e *PluginError) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	return e.Msg + ": " + e.Err.Error()
}

// Unwrap exposes the cause so errors.Is/As work.
func (e *PluginError) Unwrap() error {
	return e.Err
}

// NewPluginError builds a PluginError.
func NewPluginError(msg string, errs ...error) *PluginError {
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}

	return &PluginError{Msg: msg, Err: err}
}

// PluginErrorf builds a PluginError with formatting.
func PluginErrorf(format string, args ...any) *PluginError {
	return &PluginError{Msg: fmt.Sprintf(format, args...)}
}

// TransportError is raised when a plugin transport cannot send or receive.
type TransportError struct {
	Msg string
	Err error
}

func (e *TransportError) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	return e.Msg + ": " + e.Err.Error()
}

// Unwrap exposes the cause so errors.Is/As work.
func (e *TransportError) Unwrap() error {
	return e.Err
}

// NewTransportError builds a TransportError.
func NewTransportError(msg string, errs ...error) *TransportError {
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}

	return &TransportError{Msg: msg, Err: err}
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

func IsAria2NotFound(err error) bool {
	if err == nil {
		return false
	}
	var e *Aria2Error
	if !stdErrors.As(err, &e) {
		return false
	}

	return strings.Contains(e.Msg, "Invalid GID") || strings.Contains(e.Msg, "is not found")
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
