package api

import (
	"fmt"
	"math"
	"path/filepath"
	"rapid/lib"
	"rapid/lib/helpers"
	"reflect"
	"strings"
)

const dash = "—"

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

func (d Download) Progress() float64 {
	if d.TotalLength == 0 {
		return 0
	}
	return float64(d.CompletedLength) / float64(d.TotalLength)
}

func (d Download) FormatProgress() string {
	return fmt.Sprintf("%d%%", int(math.Round(d.Progress()*100)))
}

func (d Download) FirstPath() string {
	if len(d.Files) == 0 {
		return ""
	}
	return d.Files[0].Path
}

func (d Download) Equal(b Download) bool {
	return reflect.DeepEqual(d, b)
}

// FileDir is the directory of the first known file — what "Go to file" should
// open — falling back to the aria2 base dir when no file path is known yet.
func (d Download) FileDir() string {
	if p := d.FirstPath(); p != "" {
		return filepath.Dir(p)
	}
	return d.Dir
}

func (d Download) Name() string {
	if d.Resolved != nil {
		if d.Resolved.Title != "" {
			return d.Resolved.Title
		}
		if d.Resolved.Filename != "" {
			return d.Resolved.Filename
		}
	}
	if p := d.FirstPath(); p != "" {
		if name := filepath.Base(p); name != "" {
			return name
		}
	}
	return d.GID
}

func (d Download) FormatStatus() string {
	if d.Status == "removed" {
		return "Stopped"
	}
	if d.Status == "" {
		return ""
	}
	return strings.ToUpper(d.Status[:1]) + d.Status[1:]
}

// SpeedText is "—" on error, "<size>/s" when downloading, else "—".
func (d Download) FormatSpeed() string {
	if d.Status == "error" {
		return dash
	}
	if d.DownloadSpeed > 0 {
		return helpers.FormatSize(d.DownloadSpeed) + "/s"
	}
	return dash
}

// SizeText is "done / total", or "—" when the total is unknown.
func (d Download) FormatSize() string {
	if d.TotalLength == 0 {
		return dash
	}
	return helpers.FormatSize(d.CompletedLength) + " / " + helpers.FormatSize(d.TotalLength)
}

// CanPause is true for active/waiting downloads.
func (d Download) CanPause() bool {
	return d.Status == "active" || d.Status == "waiting"
}

// CanResume is true for paused downloads.
func (d Download) CanResume() bool {
	return d.Status == "paused"
}

// IsError reports whether the download is in the failed state.
func (d Download) IsError() bool {
	return d.Status == "error"
}

func (d Download) IsFinished() bool {
	return d.Status == "complete"
}

func (d Download) IsTerminal() bool {
	return d.Status == "error" || d.Status == "removed" || d.Status == "complete"
}

func (d Download) FormatEstimation() string {
	remaining := d.TotalLength - d.CompletedLength
	if remaining <= 0 || d.DownloadSpeed <= 0 {
		return dash
	}
	secs := float64(remaining) / float64(d.DownloadSpeed)
	if secs < 60 {
		return fmt.Sprintf("%ds", int(math.Ceil(secs)))
	}
	mins := int(math.Floor(secs / 60))
	if mins < 60 {
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%dh %dm", mins/60, mins%60)
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
