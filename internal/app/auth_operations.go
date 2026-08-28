package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
)

type AuthDependencies struct {
	Accounts       *store.AccountStore
	Devices        *store.DeviceStore
	Sessions       *store.SessionStore
	AuthService    *auth.Service
	SessionManager auth.SessionManager
	AccessSigner   auth.AccessTokenSigner
	AccessLifetime time.Duration
	Hasher         auth.PasswordHasher
	Now            func() time.Time
}

type AuthOperations struct {
	dependencies AuthDependencies
}

func NewAuthOperations(dependencies AuthDependencies) *AuthOperations {
	return &AuthOperations{dependencies: dependencies}
}

func (operations *AuthOperations) Login(ctx context.Context, input httpapi.LoginInput) (httpapi.LoginResult, error) {
	account, err := operations.dependencies.AuthService.Authenticate(ctx, input.Username, input.Password)
	if err != nil {
		return httpapi.LoginResult{}, err
	}
	fingerprint := sha256.Sum256([]byte(input.DeviceFingerprint))
	device, err := operations.dependencies.Devices.UpsertAfterPasswordLogin(ctx, account.ID, input.DeviceLabel, fingerprint[:], operations.dependencies.Now().UTC())
	if err != nil {
		return httpapi.LoginResult{}, err
	}
	credential, err := operations.dependencies.SessionManager.Issue(ctx, account.ID, device.ID, account.PasswordVersion)
	if err != nil {
		return httpapi.LoginResult{}, err
	}
	return operations.result(account, credential)
}

func (operations *AuthOperations) InitialPasswordChange(ctx context.Context, principal identity.Principal, input httpapi.PasswordChangeInput) (httpapi.PasswordChangeResult, error) {
	if !principal.Account.MustChangePassword {
		return httpapi.PasswordChangeResult{}, errors.New("account is not using an initial password")
	}
	return operations.changePassword(ctx, principal, input)
}

func (operations *AuthOperations) PasswordChange(ctx context.Context, principal identity.Principal, input httpapi.PasswordChangeInput) (httpapi.PasswordChangeResult, error) {
	if principal.Account.MustChangePassword {
		return httpapi.PasswordChangeResult{}, errors.New("initial password route is required")
	}
	return operations.changePassword(ctx, principal, input)
}

func (operations *AuthOperations) changePassword(ctx context.Context, principal identity.Principal, input httpapi.PasswordChangeInput) (httpapi.PasswordChangeResult, error) {
	account, err := operations.dependencies.AuthService.Authenticate(ctx, principal.Account.Username, input.CurrentPassword)
	if err != nil || account.ID != principal.Account.ID {
		return httpapi.PasswordChangeResult{}, auth.ErrInvalidCredentials
	}
	hash, err := operations.dependencies.Hasher.Hash(input.NewPassword, account.Username)
	if err != nil {
		return httpapi.PasswordChangeResult{}, err
	}
	revoked, err := operations.dependencies.Accounts.ChangePasswordAndRevokeSessions(ctx, account.ID, hash, operations.dependencies.Now().UTC())
	if err != nil {
		return httpapi.PasswordChangeResult{}, err
	}
	return httpapi.PasswordChangeResult{SessionsRevoked: revoked, LoginRequired: true}, nil
}

func (operations *AuthOperations) Refresh(ctx context.Context, raw string) (httpapi.LoginResult, error) {
	credential, err := operations.dependencies.SessionManager.Refresh(ctx, raw)
	if err != nil {
		return httpapi.LoginResult{}, err
	}
	account, err := operations.dependencies.Accounts.FindByID(ctx, credential.AccountID)
	if err != nil || account.DisabledAt != nil || account.PasswordVersion != credential.PasswordVersion {
		return httpapi.LoginResult{}, auth.ErrSessionRevoked
	}
	return operations.result(account, credential)
}

func (operations *AuthOperations) Logout(ctx context.Context, principal identity.Principal) error {
	return operations.dependencies.Sessions.RevokeSession(ctx, principal.Account.ID, principal.SessionID, operations.dependencies.Now().UTC())
}

func (operations *AuthOperations) result(account identity.Account, credential auth.SessionCredential) (httpapi.LoginResult, error) {
	accessToken, err := operations.dependencies.AccessSigner.Issue(auth.AccessIdentity{
		AccountID: account.ID, DeviceID: credential.DeviceID, SessionID: credential.SessionID, PasswordVersion: credential.PasswordVersion,
	})
	if err != nil {
		return httpapi.LoginResult{}, err
	}
	return httpapi.LoginResult{
		AccessToken: accessToken, RefreshToken: credential.RefreshToken,
		AccessExpiresAt:    operations.dependencies.Now().UTC().Add(operations.dependencies.AccessLifetime),
		MustChangePassword: account.MustChangePassword,
	}, nil
}

var _ httpapi.AuthOperations = (*AuthOperations)(nil)
