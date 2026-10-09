// Command seed wires db + store + seeder service (replaces
// python -m rapid.backend.seeder).
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"rapid/db"
	"rapid/services/download"
	"rapid/services/seeder"
	"rapid/services/settings"

	qt "github.com/mappu/miqt/qt6"
	"golang.org/x/sync/errgroup"
)

func main() {
	qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetApplicationName("rapid")
	base := "."
	if len(os.Args) > 1 {
		base = os.Args[1]
	}

	s := settings.Default(base)
	if err := db.Open(filepath.Join(s.DataDir, "database.db")); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGABRT, os.Interrupt, os.Kill)
	if err := run(ctx, s); err != nil {
		log.Fatal(err)
	}

	defer cancel()
}

func run(ctx context.Context, settings settings.Settings) error {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		svc := seeder.NewSeederService(
			download.NewDownloadSeeder(settings),
		)

		return svc.Seed(gctx)
	})

	return g.Wait()
}
