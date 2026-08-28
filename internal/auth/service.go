package auth

import (
	"context"
	"errors"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

var (
	ErrForbidden          = errors.New("forbidden")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountNotFound    = identity.ErrAccountNotFound
)

type AuthorizationDecision string

const (
	Allow               AuthorizationDecision = "allow"
	DenyDisabled        AuthorizationDecision = "deny_disabled"
	DenyInitialPassword AuthorizationDecision = "deny_initial_password"
)

type AccountRepository interface {
	Create(context.Context, identity.Account) (identity.Account, error)
	FindByUsername(context.Context, string) (identity.Account, error)
}

type CreateAccountInput struct {
	Username        string
	InitialPassword string
}

type Service struct {
	accounts  AccountRepository
	hasher    PasswordHasher
	dummyHash string
}

func NewService(accounts AccountRepository, hasher PasswordHasher) *Service {
	dummyHash, _ := hasher.Hash("timing probe password 2026", "timing-probe")
	return &Service{accounts: accounts, hasher: hasher, dummyHash: dummyHash}
}

func (service *Service) Authenticate(ctx context.Context, username, password string) (identity.Account, error) {
	account, err := service.accounts.FindByUsername(ctx, username)
	if errors.Is(err, identity.ErrAccountNotFound) {
		_, _ = service.hasher.Verify(password, service.dummyHash)
		return identity.Account{}, ErrInvalidCredentials
	}
	if err != nil {
		return identity.Account{}, err
	}
	valid, err := service.hasher.Verify(password, account.PasswordHash)
	if err != nil || !valid || account.DisabledAt != nil {
		return identity.Account{}, ErrInvalidCredentials
	}
	return account, nil
}

func (service *Service) Authorize(account identity.Account, capabilityID string) AuthorizationDecision {
	if account.DisabledAt != nil {
		return DenyDisabled
	}
	if account.MustChangePassword && capabilityID != "auth.password.initial_change" && capabilityID != "auth.logout" {
		return DenyInitialPassword
	}
	return Allow
}

func (service *Service) CreateAccount(ctx context.Context, actor identity.Actor, input CreateAccountInput) (identity.Account, error) {
	if !actor.PlatformAdmin {
		return identity.Account{}, ErrForbidden
	}
	hash, err := service.hasher.Hash(input.InitialPassword, input.Username)
	if err != nil {
		return identity.Account{}, err
	}
	return service.accounts.Create(ctx, identity.Account{
		Username:           input.Username,
		PasswordHash:       hash,
		PlatformAdmin:      false,
		PasswordVersion:    1,
		MustChangePassword: true,
	})
}
