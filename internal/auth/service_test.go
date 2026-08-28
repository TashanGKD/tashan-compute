package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

func TestInitialPasswordAccountCannotUseProtectedCapabilities(t *testing.T) {
	svc := NewService(nil, NewPasswordHasher(DefaultPasswordParams()))
	account := identity.Account{MustChangePassword: true}

	if decision := svc.Authorize(account, "admin.user.create"); decision != DenyInitialPassword {
		t.Fatalf("Authorize() = %q, want %q", decision, DenyInitialPassword)
	}
	if decision := svc.Authorize(account, "auth.password.initial_change"); decision != Allow {
		t.Fatalf("Authorize(initial change) = %q, want %q", decision, Allow)
	}
}

func TestAuthenticateRejectsWrongPasswordAndDisabledAccountIdentically(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())
	hash, err := hasher.Hash("very long fixture password 2026", "alice")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	repository := &memoryAccounts{created: identity.Account{Username: "alice", PasswordHash: hash}}
	svc := NewService(repository, hasher)

	if _, err := svc.Authenticate(context.Background(), "alice", "wrong fixture password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v", err)
	}
	disabledAt := time.Now()
	repository.created.DisabledAt = &disabledAt
	if _, err := svc.Authenticate(context.Background(), "alice", "very long fixture password 2026"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("disabled account error = %v", err)
	}
}

func TestOnlyPlatformAdminCanCreateOrdinaryAccount(t *testing.T) {
	repository := &memoryAccounts{}
	svc := NewService(repository, NewPasswordHasher(DefaultPasswordParams()))
	input := CreateAccountInput{Username: "alice", InitialPassword: "very long fixture password 2026"}

	_, err := svc.CreateAccount(context.Background(), identity.Actor{}, input)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("ordinary CreateAccount() error = %v", err)
	}

	created, err := svc.CreateAccount(context.Background(), identity.Actor{PlatformAdmin: true}, input)
	if err != nil {
		t.Fatalf("admin CreateAccount() error = %v", err)
	}
	if created.PlatformAdmin {
		t.Fatal("ordinary account was created as platform admin")
	}
	if !created.MustChangePassword {
		t.Fatal("created account does not require initial password change")
	}
}

type memoryAccounts struct {
	created identity.Account
}

func (repository *memoryAccounts) Create(_ context.Context, account identity.Account) (identity.Account, error) {
	repository.created = account
	return account, nil
}

func (repository *memoryAccounts) FindByUsername(_ context.Context, username string) (identity.Account, error) {
	if repository.created.Username != username {
		return identity.Account{}, ErrAccountNotFound
	}
	return repository.created, nil
}
