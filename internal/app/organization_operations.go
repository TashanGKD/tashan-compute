package app

import (
	"context"
	"errors"

	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
)

var ErrOrganizationAdminRequired = errors.New("organization administrator role is required")

type OrganizationOperations struct{ store *store.OrganizationStore }

func NewOrganizationOperations(repository *store.OrganizationStore) *OrganizationOperations {
	return &OrganizationOperations{store: repository}
}

func (operations *OrganizationOperations) CreateOrganization(ctx context.Context, principal identity.Principal, name string) (httpapi.OrganizationResult, error) {
	if !principal.Account.PlatformAdmin {
		return httpapi.OrganizationResult{}, authzForbidden()
	}
	organization, err := operations.store.Create(ctx, name, principal.Account.ID)
	return httpapi.OrganizationResult{OrganizationID: organization.ID, Name: organization.Name}, err
}

func (operations *OrganizationOperations) ListOrganizations(ctx context.Context, principal identity.Principal) ([]httpapi.OrganizationResult, error) {
	items, err := operations.store.ListForAccount(ctx, principal.Account.ID)
	if err != nil {
		return nil, err
	}
	result := make([]httpapi.OrganizationResult, 0, len(items))
	for _, item := range items {
		result = append(result, httpapi.OrganizationResult{OrganizationID: item.ID, Name: item.Name})
	}
	return result, nil
}

func (operations *OrganizationOperations) AddMember(ctx context.Context, principal identity.Principal, organizationID, accountID, role string) (httpapi.MembershipResult, error) {
	if role == "" {
		role = "developer"
	}
	return operations.changeMember(ctx, principal, identity.Membership{OrganizationID: organizationID, AccountID: accountID, Role: role})
}

func (operations *OrganizationOperations) SetMemberRole(ctx context.Context, principal identity.Principal, organizationID, accountID, role string) (httpapi.MembershipResult, error) {
	return operations.changeMember(ctx, principal, identity.Membership{OrganizationID: organizationID, AccountID: accountID, Role: role})
}

func (operations *OrganizationOperations) changeMember(ctx context.Context, principal identity.Principal, membership identity.Membership) (httpapi.MembershipResult, error) {
	if membership.Role != "org_admin" && membership.Role != "developer" && membership.Role != "viewer" {
		return httpapi.MembershipResult{}, errors.New("invalid organization role")
	}
	if !principal.Account.PlatformAdmin {
		role, err := operations.store.Role(ctx, membership.OrganizationID, principal.Account.ID)
		if err != nil || role != "org_admin" {
			return httpapi.MembershipResult{}, ErrOrganizationAdminRequired
		}
	}
	updated, err := operations.store.UpsertMember(ctx, membership)
	return httpapi.MembershipResult{OrganizationID: updated.OrganizationID, AccountID: updated.AccountID, Role: updated.Role}, err
}

func (operations *OrganizationOperations) RemoveMember(ctx context.Context, principal identity.Principal, organizationID, accountID string) error {
	if !principal.Account.PlatformAdmin {
		role, err := operations.store.Role(ctx, organizationID, principal.Account.ID)
		if err != nil || role != "org_admin" {
			return ErrOrganizationAdminRequired
		}
	}
	return operations.store.RemoveMember(ctx, organizationID, accountID)
}

func authzForbidden() error { return errors.New("platform administrator role is required") }

var _ httpapi.OrganizationOperations = (*OrganizationOperations)(nil)
