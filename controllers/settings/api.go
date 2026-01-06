package settings

type Setting struct {
	DownloadLocation string `json:"downloadLocation"`
	MaxRetry         int    `json:"maxRetry"`
	MinChunkSize     int64  `json:"minChunkSize"`
	MaxChunkCount    int    `json:"maxChunkCount"`
}
