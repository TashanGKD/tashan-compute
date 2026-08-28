package postgres_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
	"github.com/google/uuid"
)

func TestSessionStoreRotatesAtomicallyAndDetectsReplay(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	accountID, deviceID := insertAccountAndDevice(t, db)
	repository := store.NewSessionStore(db)
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	current := auth.Session{
		ID: uuid.NewString(), AccountID: accountID, DeviceID: deviceID,
		RefreshTokenHash: bytes.Repeat([]byte{0x11}, 32), FamilyID: uuid.NewString(),
		PasswordVersion: 1, ExpiresAt: now.Add(time.Hour),
	}
	if err := repository.Create(context.Background(), current); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	next := auth.Session{
		ID: uuid.NewString(), AccountID: accountID, DeviceID: deviceID,
		RefreshTokenHash: bytes.Repeat([]byte{0x22}, 32), FamilyID: current.FamilyID,
		PasswordVersion: 1, ExpiresAt: now.Add(2 * time.Hour),
	}
	if _, err := repository.Rotate(context.Background(), current.RefreshTokenHash, next, now); err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
	if _, err := repository.Rotate(context.Background(), current.RefreshTokenHash, next, now); !errors.Is(err, auth.ErrRefreshReplay) {
		t.Fatalf("Rotate(replay) error = %v", err)
	}
	if err := repository.RevokeFamily(context.Background(), current.FamilyID, now); err != nil {
		t.Fatalf("RevokeFamily() error = %v", err)
	}
	found, err := repository.FindByRefreshHash(context.Background(), next.RefreshTokenHash)
	if err != nil {
		t.Fatalf("FindByRefreshHash() error = %v", err)
	}
	if found.RevokedAt == nil {
		t.Fatal("family revoke did not revoke rotated session")
	}
}

func TestPasswordResetRevokesAllAccountSessions(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	accountID, firstDeviceID := insertAccountAndDevice(t, db)
	var secondDeviceID string
	if err := db.QueryRow(`INSERT INTO devices (account_id, label, fingerprint_hash) VALUES ($1, 'second', $2) RETURNING id`, accountID, []byte("second-fingerprint")).Scan(&secondDeviceID); err != nil {
		t.Fatalf("insert second device: %v", err)
	}
	now := time.Now().UTC()
	for index, deviceID := range []string{firstDeviceID, secondDeviceID} {
		_, err := db.Exec(`INSERT INTO sessions (account_id, device_id, refresh_token_hash, rotation_family, password_version, expires_at) VALUES ($1, $2, $3, $4, 1, $5)`, accountID, deviceID, bytes.Repeat([]byte{byte(index + 1)}, 32), uuid.NewString(), now.Add(time.Hour))
		if err != nil {
			t.Fatalf("insert session %d: %v", index, err)
		}
	}

	revoked, err := store.NewAccountStore(db).ResetPasswordAndRevokeSessions(context.Background(), accountID, "new-fixture-hash", now)
	if err != nil {
		t.Fatalf("ResetPasswordAndRevokeSessions() error = %v", err)
	}
	if revoked != 2 {
		t.Fatalf("revoked = %d, want 2", revoked)
	}
	var version int
	var mustChange bool
	if err := db.QueryRow(`SELECT password_version, must_change_password FROM accounts WHERE id = $1`, accountID).Scan(&version, &mustChange); err != nil {
		t.Fatalf("query account: %v", err)
	}
	if version != 2 || !mustChange {
		t.Fatalf("version=%d mustChange=%v", version, mustChange)
	}
}

func insertAccountAndDevice(t *testing.T, db interface {
	QueryRow(query string, args ...any) *sql.Row
}) (string, string) {
	t.Helper()
	var accountID, deviceID string
	if err := db.QueryRow(`INSERT INTO accounts (username, password_hash) VALUES ($1, 'hash') RETURNING id`, "user-"+uuid.NewString()[:8]).Scan(&accountID); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO devices (account_id, label, fingerprint_hash) VALUES ($1, 'test', $2) RETURNING id`, accountID, []byte(uuid.NewString())).Scan(&deviceID); err != nil {
		t.Fatalf("insert device: %v", err)
	}
	return accountID, deviceID
}
