package downloader

import (
	"context"
	"rapid/services/download/api"
)

// NotifyCallback fires with fresh status; ErrorCallback with a message;
// GlobalNotifyCallback with the aria2 global stat object.
type NotifyCallback func(api.Download)
type ErrorCallback func(error)
type GlobalNotifyCallback func([]byte)

type Downloader interface {
	Start(ctx context.Context) error
	Stop() error
	Download(ctx context.Context, uri api.ResolvedURL) (api.Download, error)
	Pause(ctx context.Context, id string) (api.Download, error)
	Resume(ctx context.Context, id string) (api.Download, error)
	Remove(ctx context.Context, id string) (api.Download, error)
	Purge(ctx context.Context, id string) error
	GetStatus(ctx context.Context, id string) (api.Download, error)
	Listen(id string, onNotify NotifyCallback, onError ErrorCallback)
	Unlisten(id string)
	Refresh(ctx context.Context, id ...string)
}

// Resolver turns a URI into downloadable resources
type Resolver interface {
	ShouldResolve(uri string) bool
	Resolve(ctx context.Context, uri string, options map[string]any) ([]api.ResolvedURL, error)
}
