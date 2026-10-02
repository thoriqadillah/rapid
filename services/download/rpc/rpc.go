package rpc

import "context"

type Rpc interface {
	Call(ctx context.Context, method string, body []byte) ([]byte, error)
}
