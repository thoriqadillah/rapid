package download

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"rapid/lib"
	liberrors "rapid/lib/errors"
	"rapid/services/download/api"
	"rapid/services/download/downloader"
	"rapid/services/download/plugin"
	"rapid/services/settings"
)

const echoManifest = `{
  	"id": "echo",
  	"name": "Echo",
  	"type": "resolver",
  	"match": ["https://plugin\\.example/"],
  	"transport": {
  		"command": "printf",
   		"args": ["%s\n","{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"items\":[{\"title\":\"Plugin item\",\"url\":\"https://cdn.example/p.mp4\",\"category\":\"video\"}]}}"]
    }
}`

const failManifest = `{
  	"id": "fail",
  	"name": "Fail",
  	"type": "resolver",
  	"match": ["https://fail\\.example/"],
  	"transport": {
  		"command": "printf",
   		"args": ["%s\n","{\"jsonrpc\":\"2.0\",\"id\":1,\"error\":{\"code\":-1,\"message\":\"boom\"}}"]
    }
}`

type fakeResolver struct {
	items   []api.ResolvedURL
	err     error
	calls   int
	options map[string]any
}

var _ downloader.Resolver = (*fakeResolver)(nil)

func (f *fakeResolver) ShouldResolve(string) bool { return true }

func (f *fakeResolver) Resolve(_ context.Context, _ string, options map[string]any) ([]api.ResolvedURL, error) {
	f.calls++
	f.options = options
	return f.items, f.err
}

func pluginManager(t *testing.T, manifests map[string]string) *plugin.Manager {
	t.Helper()
	root := t.TempDir()
	for name, manifest := range manifests {
		dir := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, plugin.ManifestName), []byte(manifest), 0o644))
	}
	return plugin.NewManager([]string{root})
}

func newResolveService(t *testing.T, resolver *fakeResolver, manifests map[string]string) *Service {
	t.Helper()
	s := NewService(settings.Settings{DownloadDir: t.TempDir()})
	s.resolver = resolver
	s.plugins = pluginManager(t, manifests)
	return s
}

func TestResolveUsesPluginBeforeMainResolver(t *testing.T) {
	resolver := &fakeResolver{items: []api.ResolvedURL{{URL: "https://cdn.example/fallback.mp4"}}}
	s := newResolveService(t, resolver, map[string]string{"echo": echoManifest})

	items, err := s.resolve(t.Context(), ResolveRequest{URL: "https://plugin.example/watch"})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "https://cdn.example/p.mp4", items[0].URL)
	require.Equal(t, "Echo", items[0].ResolverName)
	require.Zero(t, resolver.calls, "a matching plugin must not run the main resolver")
}

func TestResolveClaimedPluginFailureDoesNotRunMainResolver(t *testing.T) {
	resolver := &fakeResolver{items: []api.ResolvedURL{{URL: "https://cdn.example/fallback.mp4"}}}
	s := newResolveService(t, resolver, map[string]string{"fail": failManifest})

	items, err := s.resolve(t.Context(), ResolveRequest{URL: "https://fail.example/watch"})
	require.Error(t, err)
	require.Nil(t, items)
	require.Zero(t, resolver.calls, "a failed plugin claim must not trigger main resolver fallback")

	var rpcErr *liberrors.JsonRpcError
	require.ErrorAs(t, err, &rpcErr)
	require.Equal(t, -1, rpcErr.Code)
}

func TestMainResolverIsRunWhenNoPluginMatches(t *testing.T) {
	resolver := &fakeResolver{
		items: []api.ResolvedURL{{
			URL:      "https://cdn.example/fallback.mp4",
			Title:    "Fallback",
			Filename: "fallback.mp4",
			MIMEType: "video/mp4",
			Category: lib.CategoryVideo,
		}},
	}
	s := newResolveService(t, resolver, map[string]string{"echo": echoManifest})
	items, err := s.resolve(t.Context(), ResolveRequest{
		URL:      "https://other.example/watch",
		Headers:  map[string]string{"X-Test": "1"},
		Cookies:  map[string]string{"sid": "abc"},
		Referer:  "https://other.example/page",
		Title:    "Requested title",
		Category: "video",
	})
	require.NoError(t, err)
	require.Equal(t, 1, resolver.calls)
	require.Len(t, items, 1)
	require.Equal(t, "Requested title", items[0].Title, "request title wins")
	require.Equal(t, map[string]string{"X-Test": "1"}, items[0].Headers)
	require.Equal(t, map[string]string{"sid": "abc"}, items[0].Cookies)
	require.Equal(t, "https://other.example/page", items[0].Referer)
	require.Equal(t, "fallback.mp4", items[0].Filename, "empty request filename keeps the probe's")
}

func TestResolveMergesRequestContext(t *testing.T) {
	fallback := &fakeResolver{items: []api.ResolvedURL{{
		URL:      "https://cdn.example/plugin.mkv",
		Title:    "Plugin title",
		Filename: "plugin.mkv",
		MIMEType: "video/mp4",
		Referer:  "https://plugin.example/page",
		Category: lib.CategoryAudio,
	}}}
	s := newResolveService(t, fallback, nil)

	items, err := s.resolve(t.Context(), ResolveRequest{
		URL:      "https://other.example/watch",
		Filename: "request",
		Title:    "Request title",
		Category: "video",
		Headers:  map[string]string{"X-Test": "1"},
		Cookies:  map[string]string{"sid": "abc"},
	})
	require.NoError(t, err)
	require.Len(t, items, 1)

	item := items[0]
	require.Equal(t, "request.mp4", item.Filename, "request filename wins and gets the mime extension")
	require.Equal(t, "Request title", item.Title)
	require.Equal(t, lib.CategoryVideo, item.Category, "request category wins")
	require.Equal(t, "https://plugin.example/page", item.Referer, "plugin referer fills first")
	require.Equal(t, map[string]string{"X-Test": "1"}, item.Headers)
	require.Equal(t, map[string]string{"sid": "abc"}, item.Cookies)
}

func TestResolveBrowserResolvedShortCircuits(t *testing.T) {
	resolver := &fakeResolver{err: errors.New("fallback must not run")}
	s := newResolveService(t, resolver, map[string]string{"fail": failManifest})

	items, err := s.resolve(t.Context(), ResolveRequest{
		URL:             "https://fail.example/media",
		Headers:         map[string]string{"X-Test": "1"},
		Cookies:         map[string]string{"sid": "abc"},
		SavePath:        filepath.Join(t.TempDir(), "clip"),
		MIMEType:        "video/mp4",
		Size:            42,
		BrowserResolved: true,
	})
	require.NoError(t, err)
	require.Zero(t, resolver.calls)
	require.Len(t, items, 1)

	item := items[0]
	require.Equal(t, "Browser", item.ResolverName)
	require.Equal(t, "clip.mp4", item.Filename)
	require.Equal(t, "clip.mp4", item.Title)
	require.Equal(t, "video/mp4", item.MIMEType)
	require.Equal(t, lib.CategoryVideo, item.Category)
	require.Equal(t, int64(42), item.Size)
	require.Equal(t, map[string]string{"X-Test": "1"}, item.Headers)
	require.Equal(t, map[string]string{"sid": "abc"}, item.Cookies)
	require.NotEmpty(t, item.Dir)
}

func TestResolveRequiresURL(t *testing.T) {
	s := newResolveService(t, &fakeResolver{}, nil)
	_, err := s.resolve(t.Context(), ResolveRequest{URL: "  "})
	require.EqualError(t, err, "URL is required")
}

func TestDownloadUniqueNameAndContinueOnFailure(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.mkv"), []byte("x"), 0o644))

	s, fake := newActionService(t, nil)
	fake.downloadErr = map[string]error{"https://x/fail.mkv": errors.New("boom")}

	s.Download([]api.ResolvedURL{
		{URL: "https://x/fail.mkv", Title: "Fail", Filename: "a.mkv", Dir: dir},
		{URL: "https://x/ok.mkv", Title: "Ok", Filename: "b.mkv", Dir: dir},
	})
	syncQueue(t, s)

	require.Equal(t, []string{"download:https://x/fail.mkv", "download:https://x/ok.mkv"}, fake.calls)
	require.Len(t, fake.downloads, 2)
	require.Equal(t, "a (1).mkv", fake.downloads[0].Filename, "existing file forces a unique name")
	require.Equal(t, "b.mkv", fake.downloads[1].Filename)
}
