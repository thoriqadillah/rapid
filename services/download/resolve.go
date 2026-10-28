package download

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"rapid/lib"
	"rapid/lib/helpers"
	"rapid/lib/helpers/bools"
	"rapid/services/download/api"

	"github.com/mappu/miqt/qt6/mainthread"
)

// ResolveRequest is the dialog/browser input for one resolution. Headers and
// cookies are already string-filtered by the caller; Size 0 means unknown.
type ResolveRequest struct {
	URL             string
	Headers         map[string]string
	Cookies         map[string]string
	Referer         string
	PageURL         string
	Filename        string
	Title           string
	MIMEType        string
	Category        string
	SavePath        string
	Size            int64
	BrowserResolved bool
}

// browserResolvedURL builds the item the extension already resolved, filling
// the gaps from the request (filename from savePath, mime from the name, and
// the extension from the mime when the name lacks one).
func (req ResolveRequest) ToResolvedURL() api.ResolvedURL {
	filename := req.Filename
	original := filename

	dir := ""
	if req.SavePath != "" {
		dir = filepath.Dir(req.SavePath)
		filename = cmp.Or(filename, filepath.Base(req.SavePath))
	}

	mimeType := req.MIMEType
	if mimeType == "" {
		name := filename
		if name == "" {
			if parsed, err := url.Parse(req.URL); err == nil {
				name = parsed.Path
			}
		}
		mimeType = mime.TypeByExtension(path.Ext(name))
	}

	if filename != "" && mimeType != "" && filepath.Ext(filename) == "" {
		filename += helpers.ExtensionForType(mimeType)
	}

	category := lib.ToCategory(req.Category)
	if category == lib.CategoryUnknown && mimeType != "" {
		if prefix, _, ok := strings.Cut(mimeType, "/"); ok {
			switch prefix {
			case "audio", "video", "image":
				category = lib.Category(prefix)
			}
		}
	}

	title := cmp.Or(req.Title, filename)
	if title == original {
		title = filename
	}

	return api.ResolvedURL{
		URL:          req.URL,
		Title:        title,
		Filename:     filename,
		Dir:          dir,
		MIMEType:     mimeType,
		Size:         max(req.Size, 0),
		Category:     category,
		Headers:      req.Headers,
		Cookies:      req.Cookies,
		Referer:      cmp.Or(req.Referer, req.PageURL),
		ResolverName: "Browser",
	}
}

// toOptions is the plugin/fallback request context (Python passed the raw
// browser dict; plugins read headers/cookies/referer and ignore the rest).
func (r ResolveRequest) toOptions() map[string]any {
	return map[string]any{
		"headers":         r.Headers,
		"cookies":         r.Cookies,
		"referer":         r.Referer,
		"pageUrl":         r.PageURL,
		"filename":        r.Filename,
		"title":           r.Title,
		"mimeType":        r.MIMEType,
		"category":        r.Category,
		"savePath":        r.SavePath,
		"size":            r.Size,
		"browserResolved": r.BrowserResolved,
	}
}

func withRequestContext(item api.ResolvedURL, req ResolveRequest) api.ResolvedURL {
	item.Headers = bools.Ternary(item.Headers != nil, item.Headers, req.Headers)
	item.Cookies = bools.Ternary(item.Cookies != nil, item.Cookies, req.Cookies)
	item.Referer = cmp.Or(item.Referer, req.Referer, req.PageURL)
	item.Filename = cmp.Or(req.Filename, item.Filename)
	item.Title = cmp.Or(req.Title, item.Title)
	item.MIMEType = cmp.Or(item.MIMEType, req.MIMEType)
	item.Category = cmp.Or(lib.Category(req.Category), item.Category)
	if item.Filename != "" && filepath.Ext(item.Filename) == "" && item.MIMEType != "" {
		item.Filename += helpers.ExtensionForType(item.MIMEType)
	}
	return item
}

// Resolve runs the resolver chain off the GUI thread and marshals the result
// back. It is read-only, so it bypasses the action queue: an 8s plugin budget
// must not hold up a pause click.
func (s *Service) Resolve(ctx context.Context, req ResolveRequest, cb func([]api.ResolvedURL, error)) {
	go func() {
		items, err := s.resolve(ctx, req)
		mainthread.Start(func() { cb(items, err) })
	}()
}

// resolve is the Qt-free chain: browser shortcut, plugins, fallback, then
// request-context merge. A plugin error is returned as-is: never fall back
// after a plugin claimed the URI.
func (s *Service) resolve(ctx context.Context, req ResolveRequest) ([]api.ResolvedURL, error) {
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("URL is required")
	}

	if req.BrowserResolved {
		return []api.ResolvedURL{req.ToResolvedURL()}, nil
	}

	options := req.toOptions()
	items, err := s.plugins.Resolve(ctx, req.URL, options)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		items, err = s.resolver.Resolve(ctx, req.URL, options)
		if err != nil {
			return nil, err
		}
	}

	for i := range items {
		items[i] = withRequestContext(items[i], req)
	}
	return items, nil
}

// Download queues one aria2 add per item, continuing after individual
// failures. Each success re-enters the GUI thread to listen and upsert, so
// the row is live before the next poll tick.
func (s *Service) Download(items []api.ResolvedURL) {
	s.queue.Go(func(ctx context.Context) {
		var failures []error
		for _, item := range items {
			dir := cmp.Or(item.Dir, s.settings.DownloadDir)
			item.Filename = uniqueName(dir, item.Filename)

			d, err := s.downloader.Download(ctx, item)
			if err != nil {
				failures = append(failures, err)
				continue
			}

			d.Resolved = &item
			d.Category = item.Category
			s.downloader.Listen(d.GID, s.listener(d.GID))

			mainthread.Start(func() {
				s.upsertDownload(d)
			})
		}

		if len(failures) > 0 {
			mainthread.Start(func() {
				summary := fmt.Errorf("%d of %d downloads failed: %w", len(failures), len(items), errors.Join(failures...))
				s.error.Emit(summary)
			})
		}
	})
}
