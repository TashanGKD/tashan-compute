package testkit

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultTestDatabaseURL = "postgres://tcompute_test:tcompute_test_only@127.0.0.1:55432/tcompute_test?sslmode=disable"

func Postgres(t *testing.T) *sql.DB {
	t.Helper()
	databaseURL := os.Getenv("TCOMPUTE_TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultTestDatabaseURL
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open test PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		if err := db.PingContext(ctx); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("test PostgreSQL did not become ready: %v", ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
	lockConnection, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("reserve shared test PostgreSQL connection: %v", err)
	}
	if _, err := lockConnection.ExecContext(ctx, `SELECT pg_advisory_lock(827436701245199)`); err != nil {
		_ = lockConnection.Close()
		t.Fatalf("lock shared test PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		_, _ = lockConnection.ExecContext(context.Background(), `SELECT pg_advisory_unlock(827436701245199)`)
		_ = lockConnection.Close()
	})
	return db
}
