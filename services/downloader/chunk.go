package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rapid-downloader/rapid/api"
)

type Progress struct {
	Downloaded int64            `json:"downloaded"`
	Progress   float64          `json:"progress"`
	Index      int              `json:"index"`
	Size       int64            `json:"size"`
	Listener   func(p Progress) `json:"-"`
}

func newProgress(index int, size int64, listener func(p Progress)) *Progress {
	return &Progress{
		Index:    index,
		Size:     size,
		Listener: listener,
	}
}

func (p *Progress) Update(downloaded int64) {
	p.Downloaded += downloaded
	p.Progress = float64(100*p.Downloaded) / float64(p.Size)
	if p.Listener != nil {
		p.Listener(*p)
	}
}

func (p *Progress) Finish() {
	p.Downloaded = p.Size
	p.Progress = 100.0
	if p.Listener != nil {
		p.Listener(*p)
	}
}

type progressBar struct {
	reader io.ReadCloser

	progress  *Progress
	index     int
	chunkSize int64
}

func (r *progressBar) Read(payload []byte) (n int, err error) {
	n, err = r.reader.Read(payload)
	if err != nil {
		return n, err
	}

	r.progress.Update(int64(n))
	return n, err
}

func (r *progressBar) Close() error {
	return r.reader.Close()
}

type chunk struct {
	item     api.DownloadItem
	setting  api.Setting
	path     string
	index    int
	start    int64
	end      int64
	size     int64
	progress *Progress
}

func calculatePosition(item api.DownloadItem, chunkSize int64, index int) (int64, int64) {
	if item.Size == -1 {
		return -1, -1
	}

	start := int64(index * int(chunkSize))
	end := start + (chunkSize - 1)

	if index == int(item.ChunkLen)-1 {
		end = item.Size
	}

	return start, end
}

func resumePosition(location string) int64 {
	file, err := os.Stat(location)
	if err != nil {
		return 0
	}

	resumePos := file.Size()
	if err := os.Truncate(location, resumePos); err != nil {
		return 0
	}

	return resumePos
}

func newChunk(item api.DownloadItem, index int, path string, listener func(p Progress)) *chunk {
	chunkSize := item.Size / int64(item.ChunkLen)
	start, end := calculatePosition(item, chunkSize, index)

	os.MkdirAll(filepath.Join(path, item.Id), os.ModePerm)
	chunkpath := filepath.Join(path, item.Id, fmt.Sprintf("%s-%d", item.Id, index))
	progress := newProgress(index, chunkSize, listener)

	downloaded := resumePosition(chunkpath)
	start += downloaded
	progress.Update(downloaded)

	return &chunk{
		path:     chunkpath,
		item:     item,
		index:    index,
		start:    start,
		end:      end,
		size:     chunkSize,
		progress: progress,
	}
}

func (c *chunk) download(ctx context.Context) error {
	src, err := c.getDownloadFile(ctx)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := c.getSaveFile()
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	c.progress.Finish()
	return nil
}

func (c *chunk) getDownloadFile(ctx context.Context) (io.ReadCloser, error) {
	req, err := createRequest(ctx, c.item.Url, c.item.Header, c.item.Cookies)
	if err != nil {
		return nil, err
	}

	if c.start != -1 && c.end != -1 {
		bytesRange := fmt.Sprintf("bytes=%d-%d", c.start, c.end)
		req.Header.Set("Range", bytesRange)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	progressBar := &progressBar{
		reader:    res.Body,
		progress:  c.progress,
		index:     c.index,
		chunkSize: c.size,
	}

	return progressBar, nil
}

func (c *chunk) getSaveFile() (io.WriteCloser, error) {
	file, err := os.OpenFile(c.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}
