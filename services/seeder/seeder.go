// Package seeder runs data seeders (replaces python -m
// rapid.backend.seeder). Download fixtures live in backend/download/seed.go.
package seeder

import (
	"context"
	"fmt"
	"rapid/db"

	"github.com/uptrace/bun"
)

// Seeder seeds one data set.
type Seeder interface {
	Name() string
	Seed(ctx context.Context, db bun.IDB) error
}

// SeederService runs seeders in order, aborting on the first error.
type SeederService struct {
	seeders []Seeder
}

// NewSeederService builds the service.
func NewSeederService(seeders ...Seeder) *SeederService {
	return &SeederService{seeders: seeders}
}

// Seed prints "Seeding: NAME" then runs each seeder.
func (s *SeederService) Seed(ctx context.Context) error {
	return db.DB().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, seeder := range s.seeders {
			fmt.Printf("Seeding: %s\n", seeder.Name())
			if err := seeder.Seed(ctx, tx); err != nil {
				return err
			}
		}

		return nil
	})
}
