package api

import (
	"rapid/lib"
)

// SpeedSample is a point-in-time throughput record
type SpeedSample struct {
	TS    int64 `json:"ts"`
	Speed int64 `json:"speed"`
}

// FileURI is one URI candidate for a file. Empty means missing.
type FileURI struct {
	URI    string `json:"uri,omitempty"`
	Status string `json:"status,omitempty"`
}

// DownloadFile is one file inside a download
type DownloadFile struct {
	Index           int       `json:"index"`
	Path            string    `json:"path,omitempty"`
	Length          int64     `json:"length,omitempty"`
	CompletedLength int64     `json:"completedLength,omitempty"`
	Selected        bool      `json:"selected,omitempty"`
	URIs            []FileURI `json:"uris,omitempty"`
}

// Download is the aria2 tellStatus-shaped domain struct
type Download struct {
	GID             string         `json:"gid"`
	Status          string         `json:"status"`
	Dir             string         `json:"dir"`
	Category        lib.Category   `json:"category"`
	TotalLength     int64          `json:"totalLength"`
	CompletedLength int64          `json:"completedLength"`
	DownloadSpeed   int64          `json:"downloadSpeed"`
	Connections     int64          `json:"connections"`
	NumPieces       int64          `json:"numPieces"`
	PieceLength     int64          `json:"pieceLength"`
	VerifiedLength  *int64         `json:"verifiedLength,omitempty"`
	ErrorCode       int            `json:"errorCode"`
	ErrorMessage    string         `json:"errorMessage,omitempty"`
	Resolved        *ResolvedURL   `json:"resolved,omitempty"`
	Files           []DownloadFile `json:"files,omitempty"`
}

// Progress returns completed/total, or nil when total is nil or 0.
func (d Download) Progress() float64 {
	if d.TotalLength == 0 {
		return 0
	}
	p := float64(d.CompletedLength) / float64(d.TotalLength)
	return p
}

// ResolvedURL is a downloadable resource plus the request context needed to fetch it
type ResolvedURL struct {
	URL          string            `json:"url"`
	Title        string            `json:"title"`
	Filename     string            `json:"filename"`
	Dir          string            `json:"dir"`
	MIMEType     string            `json:"mimeType,omitempty"`
	Size         int64             `json:"size"`
	Category     lib.Category      `json:"category"`
	Headers      map[string]string `json:"headers,omitempty"`
	Cookies      map[string]string `json:"cookies,omitempty"`
	Referer      string            `json:"referer,omitempty"`
	ResolverName string            `json:"resolverName"`
}
