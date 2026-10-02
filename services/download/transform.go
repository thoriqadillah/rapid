package download

import (
	"rapid/lib"
	"rapid/services/download/api"
)

// Row-to-domain conversions, one per model. Ordering lives in the SQL
// (All/Get order files by index, URIs by id), so these only map.

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
	return api.FileURI{URI: u.URI, Status: u.Status}
}

// toFileURIsModel converts URI rows, already id-ordered by the query.
func toFileURIsModel(rows []fileURIrow) []api.FileURI {
	uris := make([]api.FileURI, 0, len(rows))
	for _, u := range rows {
		uris = append(uris, toFileURIModel(u))
	}
	return uris
}

// toDownloadFileModel converts one file row with its URIs.
func toDownloadFileModel(f downloadFileRow) api.DownloadFile {
	return api.DownloadFile{
		Index:           f.Index,
		Path:            f.Path,
		Length:          f.Length,
		CompletedLength: f.CompletedLength,
		Selected:        f.Selected,
		URIs:            toFileURIsModel(f.URIs),
	}
}

// toDownloadFilesModel converts file rows, already index-ordered by the query.
func toDownloadFilesModel(rows []downloadFileRow) []api.DownloadFile {
	files := make([]api.DownloadFile, 0, len(rows))
	for _, f := range rows {
		files = append(files, toDownloadFileModel(f))
	}
	return files
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
		Files:           toDownloadFilesModel(row.Files),
	}
}
