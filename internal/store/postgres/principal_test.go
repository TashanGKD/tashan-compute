package postgres_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
	"github.com/google/uuid"
)

func TestPrincipalStoreUsesCurrentServerState(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	accountID, deviceID := insertAccountAndDevice(t, db)
	sessionID := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO sessions (id, account_id, device_id, refresh_token_hash, rotation_family, password_version, expires_at) VALUES ($1, $2, $3, $4, $5, 1, $6)`, sessionID, accountID, deviceID, bytes.Repeat([]byte{0x31}, 32), uuid.NewString(), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	repository := store.NewPrincipalStore(db)
	access := auth.AccessIdentity{AccountID: accountID, DeviceID: deviceID, SessionID: sessionID, PasswordVersion: 1}

	principal, err := repository.Load(context.Background(), access)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if principal.Account.ID != accountID || principal.DeviceID != deviceID || principal.SessionID != sessionID {
		t.Fatalf("principal = %+v", principal)
	}

	if _, err := db.Exec(`UPDATE devices SET revoked_at = now() WHERE id = $1`, deviceID); err != nil {
		t.Fatalf("revoke device: %v", err)
	}
	if _, err := repository.Load(context.Background(), access); !errors.Is(err, identity.ErrPrincipalInvalid) {
		t.Fatalf("Load(revoked device) error = %v", err)
	}
}
