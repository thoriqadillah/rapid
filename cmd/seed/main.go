// Command seed wires db + store + seeder service (replaces
// python -m rapid.backend.seeder).
package main

import (
	"context"
	"log"
	"os"

	"rapid/db"
	"rapid/services/download"
	"rapid/services/seeder"
	"rapid/services/settings"
)

func main() {
	base := "."
	if len(os.Args) > 1 {
		base = os.Args[1]
	}
	s := settings.Default(base)
	if err := db.Open(s.DataDir + "/database.db"); err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	svc := seeder.NewSeederService(
		download.NewDownloadSeeder(s),
	)

	if err := svc.Seed(ctx); err != nil {
		log.Fatal(err)
	}
}
