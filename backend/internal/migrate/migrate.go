// Package migrate applies every pending SQL migration embedded in the
// migrations package (migrations/embed.go) against the target database. It
// exists so a fresh deploy — Render or anywhere else — never depends on a
// human running `migrate ... up` by hand first; cmd/server/main.go calls Up
// once at startup, before the server accepts any request.
package migrate

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/ekansh-exe/navx/migrations"
)

// Up applies every migration newer than the target database's current
// schema_migrations version. A no-op — not an error — if the schema is
// already current, so it's safe to call on every boot. Concurrent callers
// (e.g. a rolling deploy briefly running two instances) are serialized by the
// postgres driver's own advisory lock, not by anything in this function.
func Up(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("open db for migration: %w", err)
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create postgres migration driver: %w", err)
	}

	source, err := iofs.New(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
