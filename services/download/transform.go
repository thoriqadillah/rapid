package download

import (
	"rapid/lib"
	"rapid/lib/helpers/iter"
	"rapid/services/download/api"
)

// toResolvedModel returns nil for absent resolution (empty URL), so callers
// keep the nil-means-missing convention.
func toResolvedModel(r api.ResolvedURL) *api.ResolvedURL {
	if r.URL == "" {
		return nil
	}
	return &r
}

// toFileURIModel converts one URI row.
func toFileURIModel(u fileURIrow) api.FileURI {
	return api.FileURI{
		URI:    u.URI,
		Status: u.Status,
	}
}

// toDownloadFileModel converts one file row with its URIs.
func toDownloadFileModel(f downloadFileRow) api.DownloadFile {
	return api.DownloadFile{
		Index:           f.Index,
		Path:            f.Path,
		Length:          f.Length,
		CompletedLength: f.CompletedLength,
		Selected:        f.Selected,
		URIs:            iter.Map(f.URIs, toFileURIModel),
	}
}

// toDownloadModel converts a fully loaded download row.
func toDownloadModel(row downloadRow) api.Download {
	return api.Download{
		GID:             row.GID,
		Status:          row.Status,
		Dir:             row.Dir,
		Category:        lib.Category(row.Category),
		TotalLength:     row.TotalLength,
		CompletedLength: row.CompletedLength,
		DownloadSpeed:   row.DownloadSpeed,
		Connections:     row.Connections,
		NumPieces:       row.NumPieces,
		PieceLength:     row.PieceLength,
		VerifiedLength:  row.VerifiedLength,
		ErrorCode:       row.ErrorCode,
		ErrorMessage:    row.ErrorMessage,
		Resolved:        toResolvedModel(row.Resolved),
		Files:           iter.Map(row.Files, toDownloadFileModel),
	}
}
