package download

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/uptrace/bun"

	"rapid/db"
	"rapid/services/download/api"
)

func testDB(t *testing.T) *bun.DB {
	t.Helper()
	err := db.OpenMemory(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db.DB()
}

func mustUpsert(t *testing.T, db bun.IDB, d api.Download) {
	t.Helper()
	_, err := upsertDownload(t.Context(), db, d)
	require.NoError(t, err)
}

func TestStoreEmpty(t *testing.T) {
	bunDB := testDB(t)
	all, err := getAllDownloads(t.Context(), bunDB)
	require.NoError(t, err)
	require.Empty(t, all)
}

func TestStoreUpsertAndGet(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc", Status: "active"})
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Equal(t, "abc", got.GID)
	require.Equal(t, "active", got.Status)
}

func TestStoreUpsertOverwritesScalars(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc", Status: "active"})
	mustUpsert(t, bunDB, api.Download{GID: "abc", DownloadSpeed: 1024, Connections: 4})
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Empty(t, got.Status)
	require.Equal(t, int64(1024), got.DownloadSpeed)
	require.Equal(t, int64(4), got.Connections)
}

func TestStoreUpsertZeroSpeedIsKept(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc", DownloadSpeed: 1024})
	mustUpsert(t, bunDB, api.Download{GID: "abc", DownloadSpeed: 0})
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Equal(t, int64(0), got.DownloadSpeed)
}

func TestStorePersistsCategory(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc", Category: "video"})
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Equal(t, "video", string(got.Category))
}

func TestStorePersistsFilesAndUris(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{
		GID: "abc", Status: "complete", TotalLength: 100,
		Files: []api.DownloadFile{{
			Index: 1, Path: "/dl/a.bin", Length: 60,
			CompletedLength: 60, Selected: true,
			URIs: []api.FileURI{
				{URI: "http://x/a.bin", Status: "used"},
				{URI: "http://x/a.bin", Status: "waiting"},
			},
		}},
	})
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Len(t, got.Files, 1)
	require.Equal(t, 1, got.Files[0].Index)
	require.Equal(t, "/dl/a.bin", got.Files[0].Path)
	require.Len(t, got.Files[0].URIs, 2)
	require.Equal(t, "http://x/a.bin", got.Files[0].URIs[1].URI)
}

func TestStoreGetMissingReturnsZero(t *testing.T) {
	bunDB := testDB(t)
	got, err := getDownload(t.Context(), bunDB, "nope")
	require.NoError(t, err)
	require.Empty(t, got.GID)
}

func TestStorePersistsResolvedURL(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	resolved := &api.ResolvedURL{
		URL: "https://x/final.mp4", Title: "final", Filename: "final.mp4",
		MIMEType: "video/mp4", Size: 1024, Category: "video",
		Headers: map[string]string{"Referer": "https://ref.example.com"},
		Cookies: map[string]string{"session": "abc123"},
		Referer: "https://ref.example.com", ResolverName: "Rapid",
	}
	mustUpsert(t, bunDB, api.Download{GID: "abc", Status: "active", Resolved: resolved})
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.NotNil(t, got.Resolved)
	require.Equal(t, resolved.URL, got.Resolved.URL)
	require.Equal(t, "https://ref.example.com", got.Resolved.Headers["Referer"])
	require.Equal(t, "abc123", got.Resolved.Cookies["session"])
	require.Equal(t, "Rapid", got.Resolved.ResolverName)
	require.Equal(t, int64(1024), got.Resolved.Size)
}

func TestStoreMissingResolvedIsNil(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc", Status: "active"})
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Nil(t, got.Resolved)
}

func TestStoreAllNewestFirst(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "g1"})
	mustUpsert(t, bunDB, api.Download{GID: "g2"})
	all, err := getAllDownloads(ctx, bunDB)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, "g2", all[0].GID)
	require.Equal(t, "g1", all[1].GID)
}

func TestStoreRemove(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc", Status: "active"})
	require.NoError(t, removeDownload(ctx, bunDB, "abc"))
	all, err := getAllDownloads(ctx, bunDB)
	require.NoError(t, err)
	require.Empty(t, all)
}

func TestStoreClear(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "a"})
	mustUpsert(t, bunDB, api.Download{GID: "b"})
	require.NoError(t, clearDownloads(ctx, bunDB))
	all, err := getAllDownloads(ctx, bunDB)
	require.NoError(t, err)
	require.Empty(t, all)
}

func TestStoreSpeedRoundtrip(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc"})
	for _, sm := range [][2]int64{{1000, 500}, {2000, 700}, {3000, 900}} {
		require.NoError(t, addSpeedSample(ctx, bunDB, "abc", sm[0], sm[1]))
	}
	hist, err := getSpeedHistory(ctx, bunDB, "abc", 60)
	require.NoError(t, err)
	require.Equal(t, []api.SpeedSample{{TS: 1000, Speed: 500}, {TS: 2000, Speed: 700}, {TS: 3000, Speed: 900}}, hist)
}

func TestStoreSpeedLimitAndOrder(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc"})
	for i := range int64(10) {
		require.NoError(t, addSpeedSample(ctx, bunDB, "abc", i, i))
	}
	hist, err := getSpeedHistory(ctx, bunDB, "abc", 3)
	require.NoError(t, err)
	require.Len(t, hist, 3)
	require.Equal(t, int64(7), hist[0].TS)
	require.Equal(t, int64(9), hist[2].TS)
}

func TestStoreSpeedDuplicateIgnored(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc"})
	require.NoError(t, addSpeedSample(ctx, bunDB, "abc", 1000, 500))
	require.NoError(t, addSpeedSample(ctx, bunDB, "abc", 1000, 999))
	hist, err := getSpeedHistory(ctx, bunDB, "abc", 60)
	require.NoError(t, err)
	require.Equal(t, []api.SpeedSample{{TS: 1000, Speed: 500}}, hist)
}

func TestStoreRemoveCascades(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{
		GID:   "abc",
		Files: []api.DownloadFile{{Index: 0, URIs: []api.FileURI{{URI: "http://x"}}}},
	})
	require.NoError(t, addSpeedSample(ctx, bunDB, "abc", 1000, 1))
	require.NoError(t, removeDownload(ctx, bunDB, "abc"))
	hist, err := getSpeedHistory(ctx, bunDB, "abc", 60)
	require.NoError(t, err)
	require.Empty(t, hist)
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Empty(t, got.GID)
}

func TestStoreFileOverwriteClearsUnsetFields(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{
		GID:   "abc",
		Files: []api.DownloadFile{{Index: 0, Path: "/dl/a", Length: 50}},
	})
	// Second upsert overwrites the file row wholesale: omitted path clears.
	mustUpsert(t, bunDB, api.Download{
		GID: "abc",
		Files: []api.DownloadFile{{Index: 0, Length: 60,
			URIs: []api.FileURI{{URI: "http://x/a"}}}},
	})
	got, err := getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Empty(t, got.Files[0].Path)
	require.Equal(t, int64(60), got.Files[0].Length)
	require.Len(t, got.Files[0].URIs, 1)
	// Dropped index disappears.
	mustUpsert(t, bunDB, api.Download{GID: "abc", Files: []api.DownloadFile{{Index: 1}}})
	got, err = getDownload(ctx, bunDB, "abc")
	require.NoError(t, err)
	require.Len(t, got.Files, 1)
	require.Equal(t, 1, got.Files[0].Index)
}

func TestStoreUpsertReturnsFreshRow(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	row, err := upsertDownload(ctx, bunDB, api.Download{
		GID:   "abc",
		Files: []api.DownloadFile{{Index: 0, URIs: []api.FileURI{{URI: "http://x"}}}},
	})
	require.NoError(t, err)
	require.Len(t, row.Files, 1)
	require.Len(t, row.Files[0].URIs, 1)
}

func TestUpdatedAtOwnedByTrigger(t *testing.T) {
	bunDB := testDB(t)
	ctx := t.Context()
	mustUpsert(t, bunDB, api.Download{GID: "abc", Status: "active"})
	old := "2020-01-01 00:00:00"
	_, err := bunDB.NewUpdate().
		Model((*downloadRow)(nil)).
		Set("updated_at = ?", old).
		Where("gid = ?", "abc").
		Exec(ctx)
	require.NoError(t, err)
	mustUpsert(t, bunDB, api.Download{GID: "abc", Status: "paused"})
	row := new(downloadRow)
	require.NoError(t, bunDB.NewSelect().Model(row).
		Where("download_row.gid = ?", "abc").Scan(ctx))
	require.False(t, row.UpdatedAt.IsZero())
	require.NotEqual(t, "2020-01-01", row.UpdatedAt.Format("2006-01-02"),
		"trigger must have refreshed updated_at past the backdate")
}
