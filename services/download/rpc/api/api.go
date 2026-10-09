package api

import (
	"rapid/lib"
	"rapid/lib/helpers"
	"rapid/lib/helpers/iter"
	"rapid/services/download/api"
)

type JSONRpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type JSONRpcResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int           `json:"id"`
	Error   *JSONRpcError `json:"error"`
}

type Aria2URI struct {
	URI    string `json:"uri"`
	Status string `json:"status"`
}

func (s Aria2URI) ToFileURI() api.FileURI {
	return api.FileURI{
		URI:    s.URI,
		Status: s.Status,
	}
}

type Aria2File struct {
	Index           string     `json:"index"`
	Path            string     `json:"path"`
	Length          string     `json:"length"`
	CompletedLength string     `json:"completedLength"`
	Selected        string     `json:"selected"`
	URIs            []Aria2URI `json:"uris"`
}

func (s Aria2File) ToDownloadFile() api.DownloadFile {
	return api.DownloadFile{
		Index:           helpers.StringToInt[int](s.Index),
		Path:            s.Path,
		Length:          helpers.StringToInt[int64](s.Length),
		CompletedLength: helpers.StringToInt[int64](s.CompletedLength),
		Selected:        helpers.StringToBool(s.Selected),
		URIs: iter.Map(s.URIs, func(uri Aria2URI) api.FileURI {
			return uri.ToFileURI()
		}),
	}
}

type Aria2Status struct {
	GID             string      `json:"gid"`
	Status          string      `json:"status"`
	Dir             string      `json:"dir"`
	Category        string      `json:"category"`
	TotalLength     string      `json:"totalLength"`
	CompletedLength string      `json:"completedLength"`
	DownloadSpeed   string      `json:"downloadSpeed"`
	Connections     string      `json:"connections"`
	NumPieces       string      `json:"numPieces"`
	PieceLength     string      `json:"pieceLength"`
	VerifiedLength  string      `json:"verifiedLength"`
	ErrorCode       string      `json:"errorCode"`
	ErrorMessage    string      `json:"errorMessage"`
	Files           []Aria2File `json:"files"`
}

func (s Aria2Status) ToDownload() api.Download {
	return api.Download{
		GID:             s.GID,
		Status:          s.Status,
		Dir:             s.Dir,
		Category:        lib.Category(s.Category),
		TotalLength:     helpers.StringToInt[int64](s.TotalLength),
		CompletedLength: helpers.StringToInt[int64](s.CompletedLength),
		DownloadSpeed:   helpers.StringToInt[int64](s.DownloadSpeed),
		Connections:     helpers.StringToInt[int64](s.Connections),
		NumPieces:       helpers.StringToInt[int64](s.NumPieces),
		PieceLength:     helpers.StringToInt[int64](s.PieceLength),
		VerifiedLength:  helpers.StringToIntPtr[int64](s.VerifiedLength),
		ErrorCode:       helpers.StringToInt[int](s.ErrorCode),
		ErrorMessage:    s.ErrorMessage,
		Files: iter.Map(s.Files, func(f Aria2File) api.DownloadFile {
			return f.ToDownloadFile()
		}),
	}
}

type Aria2Version struct {
	Version string `json:"version"`
}

type Aria2TellStatusResponse struct {
	JSONRpcResponse
	Result Aria2Status `json:"result"`
}

type Aria2AddUriResponse struct {
	JSONRpcResponse
	Result string `json:"result"`
}

type Aria2TellActiveResponse struct {
	JSONRpcResponse
	Result []Aria2Status `json:"result"`
}

type Ari2VersionResponse struct {
	JSONRpcResponse
	Result Aria2Version `json:"result"`
}

type Aria2WebsocketGIDParam struct {
	GID string `json:"gid"`
}

type Aria2WebsocketNotification struct {
	Method string                   `json:"method"`
	Params []Aria2WebsocketGIDParam `json:"params"`
}
