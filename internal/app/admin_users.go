package app

import (
	"context"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/identity"
)

type AdminUsers struct {
	auth *auth.Service
}

func NewAdminUsers(authService *auth.Service) *AdminUsers {
	return &AdminUsers{auth: authService}
}

func (service *AdminUsers) Create(ctx context.Context, principal identity.Principal, username, initialPassword string) (httpapi.AdminUserResult, error) {
	account, err := service.auth.CreateAccount(ctx, identity.Actor{
		AccountID: principal.Account.ID, DeviceID: principal.DeviceID,
		PlatformAdmin: principal.Account.PlatformAdmin,
	}, auth.CreateAccountInput{Username: username, InitialPassword: initialPassword})
	if err != nil {
		return httpapi.AdminUserResult{}, err
	}
	return httpapi.AdminUserResult{AccountID: account.ID, MustChangePassword: account.MustChangePassword}, nil
}

var _ httpapi.AdminUserCreator = (*AdminUsers)(nil)
