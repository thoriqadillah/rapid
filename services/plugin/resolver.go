package plugin

import (
	"context"

	"rapid/lib"
	"rapid/lib/helpers/iter"
	"rapid/services/download/api"
)

// PluginResolver adapts one resolver manifest to downloader.Resolver.
type PluginResolver struct {
	manifest Manifest
}

func NewPluginResolver(manifest Manifest) *PluginResolver {
	return &PluginResolver{manifest: manifest}
}

func (r *PluginResolver) ShouldResolve(uri string) bool {
	return r.manifest.Matches(uri)
}

func (r *PluginResolver) Resolve(ctx context.Context, uri string, options map[string]any) ([]api.ResolvedURL, error) {
	items, err := runResolve(ctx, r.manifest, uri, options)
	if err != nil {
		return nil, err
	}

	return iter.Map(items, func(item ResolvedItem) api.ResolvedURL {
		return api.ResolvedURL{
			URL:          item.URL,
			Title:        item.Title,
			Filename:     item.Filename,
			MIMEType:     item.MIMEType,
			Category:     lib.ToCategory(item.Category),
			ResolverName: r.manifest.Name,
		}
	}), nil
}
