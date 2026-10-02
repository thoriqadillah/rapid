package api

import (
	"rapid/lib"
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
	uris := make([]api.FileURI, 0, len(s.URIs))
	for _, uri := range s.URIs {
		uris = append(uris, uri.ToFileURI())
	}

	return api.DownloadFile{
		Index:           lib.StringToInt[int](s.Index),
		Path:            s.Path,
		Length:          lib.StringToInt[int64](s.Length),
		CompletedLength: lib.StringToInt[int64](s.CompletedLength),
		Selected:        lib.StringToBool(s.Selected),
		URIs:            uris,
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
	files := make([]api.DownloadFile, 0, len(s.Files))
	for _, f := range s.Files {
		files = append(files, f.ToDownloadFile())
	}
	return api.Download{
		GID:             s.GID,
		Status:          s.Status,
		Dir:             s.Dir,
		Category:        lib.Category(s.Category),
		TotalLength:     lib.StringToInt[int64](s.TotalLength),
		CompletedLength: lib.StringToInt[int64](s.CompletedLength),
		DownloadSpeed:   lib.StringToInt[int64](s.DownloadSpeed),
		Connections:     lib.StringToInt[int64](s.Connections),
		NumPieces:       lib.StringToInt[int64](s.NumPieces),
		PieceLength:     lib.StringToInt[int64](s.PieceLength),
		VerifiedLength:  new(lib.StringToInt[int64](s.VerifiedLength)),
		ErrorCode:       lib.StringToInt[int](s.ErrorCode),
		ErrorMessage:    s.ErrorMessage,
		Files:           files,
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
