// Package seeder runs data seeders (replaces python -m
// rapid.backend.seeder). Download fixtures live in backend/download/seed.go.
package seeder

import (
	"context"
	"fmt"
)

type Seeder interface {
	Name() string
	Seed(ctx context.Context) error
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
	for _, seeder := range s.seeders {
		fmt.Printf("Seeding: %s\n", seeder.Name())
		if err := seeder.Seed(ctx); err != nil {
			return err
		}
	}
	return nil
}
