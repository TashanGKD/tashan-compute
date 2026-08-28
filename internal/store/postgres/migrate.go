package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/TashanGKD/tashan-compute/migrations"
)

const migrationLockID int64 = 827436701245193

func Migrate(ctx context.Context, db *sql.DB) error {
	return MigrateFS(ctx, db, migrations.Files)
}

func MigrateFS(ctx context.Context, db *sql.DB, migrationFS fs.FS) error {
	if _, err := db.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer func() {
		_, _ = db.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockID)
	}()

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename text PRIMARY KEY,
			sha256 text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		contents, err := fs.ReadFile(migrationFS, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		checksum := fmt.Sprintf("%x", sha256.Sum256(contents))

		var appliedChecksum string
		err = db.QueryRowContext(ctx, `SELECT sha256 FROM schema_migrations WHERE filename = $1`, name).Scan(&appliedChecksum)
		switch {
		case err == nil:
			if appliedChecksum != checksum {
				return fmt.Errorf("applied migration checksum changed: %s", name)
			}
			continue
		case !errors.Is(err, sql.ErrNoRows):
			return fmt.Errorf("query migration %s: %w", name, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (filename, sha256) VALUES ($1, $2)`, name, checksum); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}
	return nil
}
