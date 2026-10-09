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

func withPragmas(dsn string) string {
	if strings.Contains(dsn, "_pragma=") {
		return dsn // caller supplied its own pragmas
	}
	pragmas := []string{"foreign_keys(1)", "busy_timeout(5000)"}
	if !strings.Contains(dsn, "mode=memory") {
		pragmas = append(pragmas, "journal_mode(WAL)", "synchronous(NORMAL)")
	}
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	for _, p := range pragmas {
		dsn += sep + "_pragma=" + p
		sep = "&"
	}
	return dsn
}

func openSQL(dsn string) (*sql.DB, error) {
	sqldb, err := sql.Open(sqliteshim.ShimName, withPragmas(dsn))
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

func OpenMemory(ctx context.Context) error {
	sqldb, err := openSQL(fmt.Sprintf("file:memdb_%d?mode=memory&cache=shared", memSeq.Add(1)))
	if err != nil {
		return err
	}

	sqldb.SetMaxIdleConns(1000)
	sqldb.SetConnMaxLifetime(0)

	if err := migrate(sqldb, ctx); err != nil {
		sqldb.Close()
		return err
	}
	db = bun.NewDB(sqldb, sqlitedialect.New())
	return nil
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
