package api

type Setting struct {
	DownloadLocation string `json:"downloadLocation" toml:"download_location"`
	MaxRetry         int    `json:"maxRetry" toml:"max_retry"`
	MinChunkSize     int64  `json:"minChunkSize" toml:"min_chunk_size"`
	MaxChunkCount    int    `json:"maxChunkCount" toml:"max_chunk_count"`

	ChunkLocation string
	DataLocation  string
	LogLocation   string
}
