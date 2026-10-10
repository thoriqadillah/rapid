package download

import (
	"context"
	"database/sql"
	"slices"
	"time"

	"github.com/uptrace/bun"

	"rapid/lib/helpers/iter"
	"rapid/services/download/api"
)

// Row structs mirror the goose schema (db/migrations/00001_create_downloads.sql).
// Field types track the api models one-to-one; the store overwrites whole
// rows (merging lives in the service, which holds previous state).

type downloadRow struct {
	bun.BaseModel   `bun:"table:downloads"`
	GID             string            `bun:"gid,pk"`
	Status          string            `bun:"status"`
	Dir             string            `bun:"dir"`
	Category        string            `bun:"category"`
	TotalLength     int64             `bun:"total_length"`
	CompletedLength int64             `bun:"completed_length"`
	DownloadSpeed   int64             `bun:"download_speed"`
	Connections     int64             `bun:"connections"`
	NumPieces       int64             `bun:"num_pieces"`
	PieceLength     int64             `bun:"piece_length"`
	VerifiedLength  *int64            `bun:"verified_length"`
	ErrorCode       int               `bun:"error_code"`
	ErrorMessage    string            `bun:"error_message"`
	Resolved        api.ResolvedURL   `bun:"resolved_url"`
	CreatedAt       time.Time         `bun:"created_at,notnull"`
	UpdatedAt       time.Time         `bun:"updated_at,notnull"`
	Files           []downloadFileRow `bun:"rel:has-many,join:gid=gid"`
}

type downloadFileRow struct {
	bun.BaseModel   `bun:"table:download_files"`
	ID              int64        `bun:"id,pk,autoincrement"`
	GID             string       `bun:"gid,notnull"`
	Index           int          `bun:"index,notnull"`
	Path            string       `bun:"path"`
	Length          int64        `bun:"length"`
	CompletedLength int64        `bun:"completed_length"`
	Selected        bool         `bun:"selected"`
	URIs            []fileURIrow `bun:"rel:has-many,join:id=file_id"`
}

type fileURIrow struct {
	bun.BaseModel `bun:"table:file_uris"`
	ID            int64  `bun:"id,pk,autoincrement"`
	FileID        int64  `bun:"file_id,notnull"`
	URI           string `bun:"uri"`
	Status        string `bun:"status"`
}

type speedSampleRow struct {
	bun.BaseModel `bun:"table:speed_samples"`
	GID           string `bun:"gid,pk,notnull"`
	TS            int64  `bun:"ts,pk,notnull"`
	Speed         int64  `bun:"speed,notnull"`
}

// All returns downloads newest-first (ORDER BY created_at DESC).
func getAllDownloads(ctx context.Context, db bun.IDB) ([]api.Download, error) {
	var rows []downloadRow
	err := db.NewSelect().
		Model(&rows).
		Relation("Files", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("\"index\" ASC")
		}).
		Relation("Files.URIs", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("id ASC")
		}).
		Order("created_at DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return iter.Map(rows, toDownloadModel), nil
}

// Get returns a zero Download (and nil error) when the gid is missing.
func getDownload(ctx context.Context, db bun.IDB, gid string) (api.Download, error) {
	var row downloadRow
	err := db.NewSelect().
		Model(&row).
		Where("download_row.gid = ?", gid).
		Relation("Files", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("\"index\" ASC")
		}).
		Relation("Files.URIs", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Order("id ASC")
		}).
		Scan(ctx)
	if err == sql.ErrNoRows {
		return api.Download{}, nil
	}
	if err != nil {
		return api.Download{}, err
	}
	return toDownloadModel(row), nil
}

// AddSpeedSample inserts a sample; a duplicate ts keeps the FIRST sample.
func addSpeedSample(ctx context.Context, db bun.IDB, gid string, ts, speed int64) error {
	row := &speedSampleRow{GID: gid, TS: ts, Speed: speed}
	_, err := db.NewInsert().
		Model(row).
		On("CONFLICT (gid, ts) DO NOTHING").
		Exec(ctx)
	return err
}

// SpeedHistory returns the last limit samples in ascending ts order.
func getSpeedHistory(ctx context.Context, db bun.IDB, gid string, limit int) ([]api.SpeedSample, error) {
	if limit <= 0 {
		limit = 60
	}
	var rows []speedSampleRow
	err := db.NewSelect().Model(&rows).
		Where("speed_sample_row.gid = ?", gid).
		Order("ts DESC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]api.SpeedSample, 0, len(rows))
	for _, row := range slices.Backward(rows) {
		out = append(out, api.SpeedSample{TS: row.TS, Speed: row.Speed})
	}
	return out, nil
}

// Remove deletes one download; FK cascade clears files/uris/samples.
func removeDownload(ctx context.Context, db bun.IDB, gid string) error {
	_, err := db.NewDelete().
		Model((*downloadRow)(nil)).
		Where("gid = ?", gid).
		Exec(ctx)
	return err
}

// Clear deletes all downloads.
func clearDownloads(ctx context.Context, db bun.IDB) error {
	_, err := db.NewDelete().
		Model((*downloadRow)(nil)).
		Where("1 = 1").
		Exec(ctx)
	return err
}

// setDownloadRow overwrites every scalar from the domain struct. No presence
// checks: the service resolves merges before persisting, so by write time
// every field is final.
func setDownloadRow(row *downloadRow, dl api.Download) {
	row.Status = dl.Status
	row.Dir = dl.Dir
	row.Category = string(dl.Category)
	row.TotalLength = dl.TotalLength
	row.CompletedLength = dl.CompletedLength
	row.DownloadSpeed = dl.DownloadSpeed
	row.Connections = dl.Connections
	row.NumPieces = dl.NumPieces
	row.PieceLength = dl.PieceLength
	row.VerifiedLength = dl.VerifiedLength
	row.ErrorCode = dl.ErrorCode
	row.ErrorMessage = dl.ErrorMessage
	if dl.Resolved != nil {
		row.Resolved = *dl.Resolved
	} else {
		row.Resolved = api.ResolvedURL{}
	}
}

func upsertDownload(ctx context.Context, db bun.IDB, dl api.Download) (api.Download, error) {
	now := time.Now().UTC()
	row := new(downloadRow)
	err := db.NewSelect().
		Model(row).
		Where("download_row.gid = ?", dl.GID).
		Scan(ctx)
	if err != nil && err != sql.ErrNoRows {
		return api.Download{}, err
	}
	if err == sql.ErrNoRows {
		row = &downloadRow{
			GID:       dl.GID,
			CreatedAt: now,
			UpdatedAt: now,
			Files:     []downloadFileRow{},
		}
	}

	setDownloadRow(row, dl)
	if err == sql.ErrNoRows {
		if _, err := db.NewInsert().Model(row).Exec(ctx); err != nil {
			return api.Download{}, err
		}
	} else {
		if _, err := db.NewUpdate().
			Model(row).
			ExcludeColumn("created_at", "updated_at").
			Where("gid = ?", row.GID).
			Exec(ctx); err != nil {
			return api.Download{}, err
		}
	}

	if err := syncFiles(ctx, db, dl.GID, dl.Files); err != nil {
		return api.Download{}, err
	}

	return getDownload(ctx, db, dl.GID)
}

// loadFileRows fetches a download's file rows (with URIs) keyed by index.
func loadFileRows(ctx context.Context, db bun.IDB, gid string) (map[int]downloadFileRow, error) {
	var existing []downloadFileRow
	if err := db.NewSelect().
		Model(&existing).
		Where("download_file_row.gid = ?", gid).
		Relation("URIs").
		Scan(ctx); err != nil {
		return nil, err
	}

	return iter.KeyBy(existing, func(dfr downloadFileRow) int { return dfr.Index }), nil
}

// insertFileRow creates a file row, returning it with its new id.
func insertFileRow(ctx context.Context, db bun.IDB, gid string, f api.DownloadFile) (downloadFileRow, error) {
	fr := downloadFileRow{
		GID:             gid,
		Index:           f.Index,
		Path:            f.Path,
		Length:          f.Length,
		CompletedLength: f.CompletedLength,
		Selected:        f.Selected,
	}
	if _, err := db.NewInsert().
		Model(&fr).
		Returning("id").
		Exec(ctx); err != nil {
		return downloadFileRow{}, err
	}
	return fr, nil
}

// updateFileRow overwrites the file row wholesale, then drops the stale URI
// rows (rebuilt right after by the caller).
func updateFileRow(ctx context.Context, db bun.IDB, fr downloadFileRow, f api.DownloadFile) error {
	patch := downloadFileRow{
		ID:              fr.ID,
		Path:            f.Path,
		Length:          f.Length,
		CompletedLength: f.CompletedLength,
		Selected:        f.Selected,
	}

	if _, err := db.NewUpdate().
		Model(&patch).
		ExcludeColumn("gid", "index").
		Where("id = ?", fr.ID).
		Exec(ctx); err != nil {
		return err
	}
	_, err := db.NewDelete().
		Model((*fileURIrow)(nil)).
		Where("file_id = ?", fr.ID).
		Exec(ctx)
	return err
}

// insertFileURIs writes a file's URI rows in stored order with a single
// bulk INSERT (SQLite assigns rowids in VALUES order, so id-ordered reads
// come back in this same order).
func insertFileURIs(ctx context.Context, db bun.IDB, fileID int64, uris []api.FileURI) error {
	if len(uris) == 0 {
		return nil // bun rejects empty bulk inserts; nothing to write anyway
	}

	rows := iter.Map(uris, func(u api.FileURI) fileURIrow {
		return fileURIrow{
			FileID: fileID,
			URI:    u.URI,
			Status: u.Status,
		}
	})

	_, err := db.NewInsert().Model(&rows).Exec(ctx)
	return err
}

// pruneStaleFiles deletes file rows whose index vanished from the payload
// (all of them when the payload carries no files).
func pruneStaleFiles(ctx context.Context, db bun.IDB, gid string, seen map[int]bool) error {
	if len(seen) == 0 {
		_, err := db.NewDelete().
			Model((*downloadFileRow)(nil)).
			Where("gid = ?", gid).
			Exec(ctx)
		return err
	}
	indexes := make([]int, 0, len(seen))
	for idx := range seen {
		indexes = append(indexes, idx)
	}

	_, err := db.NewDelete().
		Model((*downloadFileRow)(nil)).
		Where("gid = ?", gid).
		Where("\"index\" NOT IN (?)", bun.List(indexes)).
		Exec(ctx)
	return err
}

// syncFiles reconciles a download's file + URI rows with the payload:
// insert-or-overwrite per index, rebuild URIs per file, prune dropped indexes.
func syncFiles(ctx context.Context, db bun.IDB, gid string, files []api.DownloadFile) error {
	byIndex, err := loadFileRows(ctx, db, gid)
	if err != nil {
		return err
	}
	seen := make(map[int]bool, len(files))
	for _, f := range files {
		seen[f.Index] = true
		var fileID int64
		if fr, ok := byIndex[f.Index]; ok {
			if err := updateFileRow(ctx, db, fr, f); err != nil {
				return err
			}
			fileID = fr.ID
		} else {
			newFr, err := insertFileRow(ctx, db, gid, f)
			if err != nil {
				return err
			}
			fileID = newFr.ID
		}
		if err := insertFileURIs(ctx, db, fileID, f.URIs); err != nil {
			return err
		}
	}
	return pruneStaleFiles(ctx, db, gid, seen)
}
