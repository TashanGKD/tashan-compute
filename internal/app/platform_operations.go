package app

import (
	"context"
	"errors"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
)

type PlatformOperations struct {
	accounts      *store.AccountStore
	hasher        auth.PasswordHasher
	organizations *OrganizationOperations
	now           func() time.Time
}

func NewPlatformOperations(accounts *store.AccountStore, hasher auth.PasswordHasher, organizations *OrganizationOperations, now func() time.Time) *PlatformOperations {
	return &PlatformOperations{accounts: accounts, hasher: hasher, organizations: organizations, now: now}
}

func (operations *PlatformOperations) ResetUserPassword(ctx context.Context, principal identity.Principal, accountID, initialPassword string) (httpapi.PasswordChangeResult, error) {
	if !principal.Account.PlatformAdmin {
		return httpapi.PasswordChangeResult{}, errors.New("platform administrator role is required")
	}
	account, err := operations.accounts.FindByID(ctx, accountID)
	if err != nil {
		return httpapi.PasswordChangeResult{}, err
	}
	hash, err := operations.hasher.Hash(initialPassword, account.Username)
	if err != nil {
		return httpapi.PasswordChangeResult{}, err
	}
	revoked, err := operations.accounts.ResetPasswordAndRevokeSessions(ctx, accountID, hash, operations.now().UTC())
	return httpapi.PasswordChangeResult{SessionsRevoked: revoked, LoginRequired: true}, err
}

func (operations *PlatformOperations) DisableUser(ctx context.Context, principal identity.Principal, accountID string) error {
	if !principal.Account.PlatformAdmin {
		return errors.New("platform administrator role is required")
	}
	_, err := operations.accounts.DisableAndRevokeSessions(ctx, accountID, operations.now().UTC())
	return err
}

func (operations *PlatformOperations) CreateOrganization(ctx context.Context, principal identity.Principal, name string) (httpapi.OrganizationResult, error) {
	return operations.organizations.CreateOrganization(ctx, principal, name)
}

var _ httpapi.PlatformOperations = (*PlatformOperations)(nil)
