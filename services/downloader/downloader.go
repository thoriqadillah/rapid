package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rapid-downloader/rapid/api"
	"github.com/rapid-downloader/rapid/services/settings"
	"golang.org/x/sync/errgroup"
)

var (
	ErrExpired = errors.New("url expired")
)

type options struct {
	id      string
	headers http.Header
	cookies []*http.Cookie
}

type Option func(o *options)

func WithId(id string) Option {
	return func(o *options) {
		o.id = id
	}
}

func WithHeaders(h http.Header) Option {
	return func(o *options) {
		o.headers = h
	}
}

func WithCookies(c []*http.Cookie) Option {
	return func(o *options) {
		o.cookies = c
	}
}

func FromItem(item api.DownloadItem) Option {
	return func(o *options) {
		o.cookies = item.Cookies
		o.headers = item.Header
		o.id = item.Id
	}
}

type Downloader interface {
	Fetch(ctx context.Context, url string, opts ...Option) (api.DownloadItem, error)
	Resume(ctx context.Context, item api.DownloadItem, listener func(p Progress)) error
	Restart(ctx context.Context, item api.DownloadItem, listener func(p Progress)) error
	Download(ctx context.Context, item api.DownloadItem, listener func(p Progress)) error
	Stop(id string) error
	Pause(id string) error
}

type DownloaderService struct {
	settingSvc *settings.SettingService
	cancelMap  map[string]func()
}

func NewService(settingSvc *settings.SettingService) *DownloaderService {
	return &DownloaderService{
		settingSvc: settingSvc,
		cancelMap:  make(map[string]func()),
	}
}

func (s *DownloaderService) context(parent context.Context, id string) context.Context {
	s.cancel(id)
	ctx, cancel := context.WithCancel(parent)
	s.cancelMap[id] = cancel
	return ctx
}

func (s *DownloaderService) cancel(id string) {
	if cancel, ok := s.cancelMap[id]; ok {
		cancel()
		delete(s.cancelMap, id)
	}
}

func (s *DownloaderService) Fetch(ctx context.Context, url string, opts ...Option) (api.DownloadItem, error) {
	setting, err := s.settingSvc.Load()
	if err != nil {
		return api.DownloadItem{}, err
	}

	options := &options{
		id:      randomId(5),
		headers: make(http.Header),
		cookies: make([]*http.Cookie, 0),
	}

	for _, opt := range opts {
		opt(options)
	}

	req, err := createRequest(ctx, url, options.headers, options.cookies)
	if err != nil {
		return api.DownloadItem{}, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return api.DownloadItem{}, err
	}

	if res.StatusCode != http.StatusOK {
		return api.DownloadItem{}, fmt.Errorf("error fetching url: %s", res.Status)
	}

	if res.ContentLength <= 0 {
		return api.DownloadItem{}, ErrExpired
	}

	resumable := resumable(res)
	filename := filepath.Base(handleDuplicate(filepath.Join(setting.DownloadLocation, filename(res))))
	location := filepath.Join(setting.DownloadLocation, filename)
	filetype := filetype(filename)
	chunklen := calculatePartition(res.ContentLength, setting)

	if !resumable {
		chunklen = 1
	}

	size := res.ContentLength
	item := api.DownloadItem{
		Id:        options.id,
		Filename:  filename,
		Filepath:  location,
		Filetype:  filetype,
		Url:       res.Request.URL.String(),
		Size:      size,
		ChunkLen:  chunklen,
		Resumable: resumable,
	}

	return item, nil
}

func (s *DownloaderService) refreshUrl(ctx context.Context, item api.DownloadItem) (api.DownloadItem, error) {
	item, err := s.Fetch(ctx, item.Url, FromItem(item))
	if err != nil {
		return item, err
	}

	return item, nil
}

func (s *DownloaderService) Download(ctx context.Context, item api.DownloadItem, listener func(p Progress)) error {
	setting, err := s.settingSvc.Load()
	if err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(s.context(ctx, item.Id))
	group.SetLimit(item.ChunkLen)

	for i := range item.ChunkLen {
		group.Go(func() error {
			chunk := newChunk(item, i, setting.ChunkLocation, listener)
			return chunk.download(groupCtx)
		})
	}

	if err := group.Wait(); err != nil {
		return err
	}

	if err := s.createFile(item); err != nil {
		return err
	}

	return s.Stop(item.Id)
}

func (s *DownloaderService) Stop(id string) error {
	s.cancel(id)
	setting, err := s.settingSvc.Load()
	if err != nil {
		return err
	}

	return os.RemoveAll(filepath.Join(setting.ChunkLocation, id))
}

func (s *DownloaderService) Pause(id string) error {
	s.cancel(id)
	return nil
}

func (s *DownloaderService) Resume(ctx context.Context, item api.DownloadItem, listener func(p Progress)) error {
	newItem, err := s.refreshUrl(ctx, item)
	if err != nil {
		return ErrExpired
	}

	return s.Download(ctx, newItem, listener)
}

func (s *DownloaderService) Restart(ctx context.Context, item api.DownloadItem, listener func(p Progress)) error {
	if err := s.Stop(item.Id); err != nil {
		return err
	}

	newItem, err := s.refreshUrl(ctx, item)
	if err != nil {
		return ErrExpired
	}

	return s.Download(ctx, newItem, listener)
}

func (s *DownloaderService) createFile(item api.DownloadItem) error {
	setting, err := s.settingSvc.Load()
	if err != nil {
		return err
	}

	file, err := os.Create(item.Filepath)
	if err != nil {
		return err
	}

	defer file.Close()

	for i := range item.ChunkLen {
		tmpFilename := filepath.Join(setting.ChunkLocation, item.Id, fmt.Sprintf("%s-%d", item.Id, i))
		if err := s.appendChunk(file, tmpFilename); err != nil {
			return err
		}
	}

	return nil
}

func (s *DownloaderService) appendChunk(dst io.Writer, srcName string) error {
	tmpFile, err := os.Open(srcName)
	if err != nil {
		return err
	}

	defer tmpFile.Close()

	if _, err := io.Copy(dst, tmpFile); err != nil {
		return err
	}

	return nil
}
