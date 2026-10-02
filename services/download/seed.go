package download

import (
	"context"
	"math"
	"math/rand/v2"
	"time"

	"rapid/lib"
	"rapid/services/download/api"
	"rapid/services/settings"

	"github.com/uptrace/bun"
)

func resolved(url, title string, category lib.Category, size int64) *api.ResolvedURL {
	return &api.ResolvedURL{
		URL:      url,
		Title:    title,
		Filename: title,
		Category: category,
		Size:     size,
	}
}

func file(index int, p string, length, completed int64) api.DownloadFile {
	return api.DownloadFile{
		Index:           index,
		Path:            p,
		Length:          length,
		CompletedLength: completed,
		Selected:        true,
		URIs: []api.FileURI{{
			URI:    "https://mirror.example.com/" + p,
			Status: "used",
		}},
	}
}

// speedCurve is a realistic history: warm-up ramp, bursts, stalls, jitter.
func speedCurve(base int64, samples int, seed uint64) []int64 {
	rng := rand.New(rand.NewPCG(seed, 0))
	curve := make([]int64, 0, samples)
	for i := range samples {
		t := float64(i) / float64(samples)
		ramp := min(1.0, t*4)
		burst := 0.25*math.Sin(float64(i)/9) + 0.15*math.Sin(float64(i)/3.7)
		stall := 0.9
		if rng.Float64() <= 0.05 {
			stall = 0.15
		}
		jitter := 0.8 + rng.Float64()*0.35
		speed := max(0, int64(float64(base)*ramp*stall*(1+burst)*jitter))
		curve = append(curve, speed)
	}
	return curve
}

// samples returns the 7 fixtures with their canonical gids/statuses.
func samples(dlDir string) []api.Download {
	return []api.Download{
		{
			GID:    "d6b8a91c2e5f4a7b",
			Status: "active",
			Dir:    dlDir, Category: "video",
			Resolved:        resolved("https://example.com/movies/interstellar.mkv", "interstellar.mkv", "video", 2147483648),
			TotalLength:     2147483648,
			CompletedLength: 1431655765,
			DownloadSpeed:   2500000,
			Connections:     8,
			NumPieces:       2048,
			PieceLength:     1048576,
			VerifiedLength:  new(int64(1431655765)),
			Files: []api.DownloadFile{
				file(0, "movies/interstellar.mkv", 2147483648, 1431655765),
				file(1, "movies/interstellar.srt", 45678, 45678),
			},
		},
		{
			GID:    "9c3d7e1a4b5f6c2d",
			Status: "complete",
			Dir:    dlDir, Category: "video",
			Resolved:        resolved("https://example.com/movies/the-dark-knight.mkv", "the-dark-knight.mkv", "video", 894784853),
			TotalLength:     894784853,
			CompletedLength: 894784853,
			DownloadSpeed:   0,
			Connections:     0,
			NumPieces:       853,
			PieceLength:     1048576,
			VerifiedLength:  new(int64(894784853)),
			Files: []api.DownloadFile{
				file(0, "movies/the-dark-knight.mkv", 894784853, 894784853),
			},
		},
		{
			GID:             "5f0a2b8c3d1e4f6a",
			Status:          "paused",
			Dir:             dlDir,
			Category:        "audio",
			Resolved:        resolved("https://example.com/music/album.zip", "album.zip", "audio", 104857600),
			TotalLength:     104857600,
			CompletedLength: 31457280,
			DownloadSpeed:   0,
			Connections:     4,
			NumPieces:       100,
			PieceLength:     1048576,
			VerifiedLength:  new(int64(31457280)),
			Files: []api.DownloadFile{
				file(0, "music/album.zip", 104857600, 31457280),
			},
		},
		{
			GID:             "a1b2c3d4e5f60718",
			Status:          "active",
			Dir:             dlDir,
			Category:        "compressed",
			Resolved:        resolved("https://example.com/archives/linux-kernel-6.12.tar.xz", "linux-kernel-6.12.tar.xz", "compressed", 4294967296),
			TotalLength:     4294967296,
			CompletedLength: 3221225472,
			DownloadSpeed:   8400000,
			Connections:     24,
			NumPieces:       4096,
			PieceLength:     1048576,
			VerifiedLength:  new(int64(3221225472)),
			Files: []api.DownloadFile{
				file(0, "archives/linux-kernel-6.12.tar.xz", 4294967296, 3221225472),
			},
		},
		{
			GID:             "e7f8091a2b3c4d5e",
			Status:          "error",
			Dir:             dlDir,
			Category:        "document",
			Resolved:        resolved("https://example.com/docs/manual.pdf", "manual.pdf", "document", 5242880),
			TotalLength:     5242880,
			CompletedLength: 1048576,
			DownloadSpeed:   0,
			Connections:     0,
			ErrorCode:       1,
			ErrorMessage:    "URI not found",
			Files: []api.DownloadFile{
				file(0, "docs/manual.pdf", 5242880, 1048576),
			},
		},
		{
			GID:             "1122334455667788",
			Status:          "complete",
			Dir:             dlDir,
			Category:        "application",
			Resolved:        resolved("https://example.com/software/setup.bin", "setup.bin", "application", 734003200),
			TotalLength:     734003200,
			CompletedLength: 734003200,
			DownloadSpeed:   0,
			Connections:     4,
			Files: []api.DownloadFile{
				file(0, "software/setup.bin", 734003200, 734003200),
			},
		},
		{
			GID:             "f1e2d3c4b5a60719",
			Status:          "waiting",
			Dir:             dlDir,
			Category:        "document",
			Resolved:        resolved("https://example.com/books/clean-code.pdf", "clean-code.pdf", "document", 25165824),
			TotalLength:     25165824,
			CompletedLength: 0,
			DownloadSpeed:   0,
			Connections:     0,
			Files: []api.DownloadFile{
				file(0, "books/clean-code.pdf", 25165824, 0),
			},
		},
	}
}

type DownloadSeeder struct {
	settings settings.Settings
}

func NewDownloadSeeder(s settings.Settings) *DownloadSeeder {
	return &DownloadSeeder{
		settings: s,
	}
}

func (d *DownloadSeeder) Name() string {
	return "DownloadSeeder"
}

func (d *DownloadSeeder) Seed(ctx context.Context, db bun.IDB) error {
	for i, dl := range samples(d.settings.DownloadDir) {
		_, err := upsertDownload(ctx, db, dl)
		if err != nil {
			return err
		}

		if dl.Status == "active" {
			base := dl.DownloadSpeed
			if base == 0 {
				base = 1000000
			}
			now := time.Now().UnixMilli()
			for j, speed := range speedCurve(base, 60, uint64(i)) {
				timestamp := now - 60000 + int64(j)*1000
				if err := addSpeedSample(ctx, db, dl.GID, timestamp, speed); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
