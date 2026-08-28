package postgres_test

import (
	"context"
	"errors"
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

func TestAccountStoreBootstrapsExactlyOnePlatformAdministrator(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	repository := store.NewAccountStore(db)
	account, err := repository.Bootstrap(context.Background(), identity.Account{
		Username: "root", PasswordHash: "fixture-hash", PasswordVersion: 1,
	})
	if err != nil {
		t.Fatalf("first Bootstrap() error = %v", err)
	}
	if !account.PlatformAdmin {
		t.Fatal("bootstrapped account is not platform administrator")
	}
	if _, err := repository.Bootstrap(context.Background(), identity.Account{
		Username: "other", PasswordHash: "fixture-hash", PasswordVersion: 1,
	}); !errors.Is(err, identity.ErrBootstrapCompleted) {
		t.Fatalf("second Bootstrap() error = %v", err)
	}
}
