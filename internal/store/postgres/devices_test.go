package postgres_test

import (
	"context"
	"testing"
	"time"

	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
)

func TestDeviceStoreIsAccountScopedAndReactivatesOnlyAfterPasswordLogin(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	accountID, _ := insertAccountAndDevice(t, db)
	otherAccountID, _ := insertAccountAndDevice(t, db)
	repository := store.NewDeviceStore(db)
	now := time.Now().UTC()

	device, err := repository.UpsertAfterPasswordLogin(context.Background(), accountID, "laptop", []byte("fingerprint"), now)
	if err != nil {
		t.Fatalf("UpsertAfterPasswordLogin() error = %v", err)
	}
	if err := repository.Revoke(context.Background(), otherAccountID, device.ID, now); err == nil {
		t.Fatal("other account revoked device")
	}
	if err := repository.Revoke(context.Background(), accountID, device.ID, now); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	reactivated, err := repository.UpsertAfterPasswordLogin(context.Background(), accountID, "laptop renamed", []byte("fingerprint"), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("reactivate error = %v", err)
	}
	if reactivated.ID != device.ID || reactivated.RevokedAt != nil {
		t.Fatalf("reactivated = %+v", reactivated)
	}
}
