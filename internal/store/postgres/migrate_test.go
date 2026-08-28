package postgres_test

import (
	"context"
	"database/sql"
	"testing"
	"testing/fstest"

	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
)

func TestIdentitySchemaEnforcesServerRoles(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	for _, table := range []string{"accounts", "devices", "sessions", "organizations", "memberships", "audit_events"} {
		if !tableExists(t, db, table) {
			t.Fatalf("table %s does not exist", table)
		}
	}

	var accountID, organizationID string
	if err := db.QueryRow(`INSERT INTO accounts (username, password_hash) VALUES ('alice', 'hash') RETURNING id`).Scan(&accountID); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO organizations (name, created_by) VALUES ('example', $1) RETURNING id`, accountID).Scan(&organizationID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO memberships (organization_id, account_id, role) VALUES ($1, $2, 'client_supplied_admin')`, organizationID, accountID); err == nil {
		t.Fatal("invalid client-supplied role was accepted")
	}
}

func TestMigrationChecksumRejectsChangedAppliedFile(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	first := fstest.MapFS{"0001.sql": {Data: []byte(`CREATE TABLE example (id integer primary key);`)}}
	if err := store.MigrateFS(context.Background(), db, first); err != nil {
		t.Fatalf("first MigrateFS() error = %v", err)
	}
	changed := fstest.MapFS{"0001.sql": {Data: []byte(`CREATE TABLE example (id bigint primary key);`)}}
	if err := store.MigrateFS(context.Background(), db, changed); err == nil {
		t.Fatal("changed applied migration was accepted")
	}
}

func resetPublicSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatalf("reset public schema: %v", err)
	}
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRow(`SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil {
		t.Fatalf("tableExists(%s): %v", table, err)
	}
	return exists
}
