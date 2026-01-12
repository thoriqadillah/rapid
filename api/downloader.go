package api

import "net/http"

type DownloadItem struct {
	http.Header
	Cookies []*http.Cookie

	Id        string `json:"id"`
	Resumable bool   `json:"resumable"`
	Filename  string `json:"filename"`
	Filetype  string `json:"filetype"`
	Filepath  string `json:"filepath"`
	Url       string `json:"url"`
	Size      int64  `json:"size"`
	ChunkLen  int    `json:"chunklen"`
}
