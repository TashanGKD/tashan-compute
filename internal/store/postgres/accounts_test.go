package postgres_test

import (
	"context"
	"testing"

	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
)

func TestAccountStoreCreatesOrdinaryInitialPasswordAccount(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	repository := store.NewAccountStore(db)

	account, err := repository.Create(context.Background(), identity.Account{
		Username:           "alice",
		PasswordHash:       "fixture-hash",
		PasswordVersion:    1,
		MustChangePassword: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if account.ID == "" {
		t.Fatal("Create() returned empty ID")
	}
	if account.PlatformAdmin {
		t.Fatal("Create() returned platform admin")
	}
	found, err := repository.FindByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if found.ID != account.ID || found.PasswordHash != "fixture-hash" {
		t.Fatalf("FindByUsername() = %+v", found)
	}
}
