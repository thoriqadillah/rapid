package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

var db *bun.DB

// withFK ensures PRAGMA foreign_keys=ON on every pooled connection.
// modernc sqlite (via sqliteshim) applies _pragma params per connection,
// which covers pooled conns that a one-time Exec would miss.
func withFK(dsn string) string {
	if strings.Contains(dsn, "_pragma=") {
		return dsn
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "_pragma=foreign_keys(1)"
}

func openSQL(dsn string) (*sql.DB, error) {
	sqldb, err := sql.Open(sqliteshim.ShimName, withFK(dsn))
	if err != nil {
		return nil, err
	}

	return sqldb, nil
}

func migrate(sqldb *sql.DB, ctx context.Context) error {
	goose.SetBaseFS(migrationFS)
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.UpContext(ctx, sqldb, "migrations")
}

func Open(dsn string) error {
	sqldb, err := openSQL(dsn)
	if err != nil {
		return err
	}

	if err := migrate(sqldb, context.Background()); err != nil {
		sqldb.Close()
		return err
	}

	db = bun.NewDB(sqldb, sqlitedialect.New())
	return nil
}

var memSeq atomic.Uint64

// OpenMemory returns a bun DB on a private in-memory sqlite instance with
// migrations applied. Intended for tests; caller closes via db.Close().
func OpenMemory(ctx context.Context) (*bun.DB, error) {
	sqldb, err := openSQL(fmt.Sprintf("file:memdb_%d?mode=memory&cache=shared", memSeq.Add(1)))
	if err != nil {
		return nil, err
	}

	sqldb.SetMaxIdleConns(1000)
	sqldb.SetConnMaxLifetime(0)

	if err := migrate(sqldb, ctx); err != nil {
		sqldb.Close()
		return nil, err
	}
	return bun.NewDB(sqldb, sqlitedialect.New()), nil
}

func Close() error {
	if db == nil {
		return nil
	}
	err := db.Close()
	db = nil
	return err
}

func DB() *bun.DB {
	return db
}
