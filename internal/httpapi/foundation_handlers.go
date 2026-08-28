package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/TashanGKD/tashan-compute/internal/capability"
	"github.com/TashanGKD/tashan-compute/internal/identity"
)

type DeviceOperations interface {
	ListDevices(context.Context, Principal) ([]identity.Device, error)
	RevokeDevice(context.Context, Principal, string) error
}

type PlatformOperations interface {
	ResetUserPassword(context.Context, Principal, string, string) (PasswordChangeResult, error)
	DisableUser(context.Context, Principal, string) error
	CreateOrganization(context.Context, Principal, string) (OrganizationResult, error)
}

type OrganizationResult struct {
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
}

type MembershipResult struct {
	OrganizationID string `json:"organization_id"`
	AccountID      string `json:"account_id"`
	Role           string `json:"role"`
}

type OrganizationOperations interface {
	ListOrganizations(context.Context, Principal) ([]OrganizationResult, error)
	AddMember(context.Context, Principal, string, string, string) (MembershipResult, error)
	RemoveMember(context.Context, Principal, string, string) error
	SetMemberRole(context.Context, Principal, string, string, string) (MembershipResult, error)
}

type AuditOperations interface {
	ListAudit(context.Context, Principal) ([]map[string]any, error)
}

func capabilitiesHandler(items []capability.Capability) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		if items == nil {
			items = []capability.Capability{}
		}
		writeJSON(writer, http.StatusOK, map[string]any{"capabilities": items})
	}
}

func devicesHandler(authenticator Authenticator, operations DeviceOperations) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := requirePrincipal(writer, request, authenticator, false)
		if !ok {
			return
		}
		if operations == nil {
			serviceUnavailable(writer, request)
			return
		}
		devices, err := operations.ListDevices(request.Context(), principal)
		if err != nil {
			WriteError(writer, http.StatusBadRequest, "device.list_failed", "devices could not be listed", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"devices": devices})
	}
}

func deviceRevokeHandler(authenticator Authenticator, operations DeviceOperations) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := requirePrincipal(writer, request, authenticator, false)
		if !ok {
			return
		}
		if operations == nil {
			serviceUnavailable(writer, request)
			return
		}
		if err := operations.RevokeDevice(request.Context(), principal, request.PathValue("deviceID")); err != nil {
			WriteError(writer, http.StatusBadRequest, "device.revoke_failed", "device could not be revoked", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]bool{"revoked": true})
	}
}

func adminResetPasswordHandler(authenticator Authenticator, operations PlatformOperations) http.HandlerFunc {
	type input struct {
		InitialPassword string `json:"initial_password"`
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := requirePrincipal(writer, request, authenticator, true)
		if !ok {
			return
		}
		var body input
		if err := DecodeJSON(writer, request, &body); err != nil {
			return
		}
		if operations == nil || body.InitialPassword == "" {
			serviceUnavailable(writer, request)
			return
		}
		result, err := operations.ResetUserPassword(request.Context(), principal, request.PathValue("accountID"), body.InitialPassword)
		if err != nil {
			WriteError(writer, http.StatusBadRequest, "admin.user.reset_failed", "password could not be reset", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	}
}

func adminDisableUserHandler(authenticator Authenticator, operations PlatformOperations) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := requirePrincipal(writer, request, authenticator, true)
		if !ok {
			return
		}
		if operations == nil {
			serviceUnavailable(writer, request)
			return
		}
		if err := operations.DisableUser(request.Context(), principal, request.PathValue("accountID")); err != nil {
			WriteError(writer, http.StatusBadRequest, "admin.user.disable_failed", "account could not be disabled", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]bool{"disabled": true})
	}
}

func adminCreateOrganizationHandler(authenticator Authenticator, operations PlatformOperations) http.HandlerFunc {
	type input struct {
		Name string `json:"name"`
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := requirePrincipal(writer, request, authenticator, true)
		if !ok {
			return
		}
		var body input
		if err := DecodeJSON(writer, request, &body); err != nil {
			return
		}
		if operations == nil || body.Name == "" {
			serviceUnavailable(writer, request)
			return
		}
		result, err := operations.CreateOrganization(request.Context(), principal, body.Name)
		if err != nil {
			WriteError(writer, http.StatusBadRequest, "admin.organization.create_failed", "organization could not be created", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusCreated, result)
	}
}

func organizationsHandler(authenticator Authenticator, operations OrganizationOperations) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := requirePrincipal(writer, request, authenticator, false)
		if !ok {
			return
		}
		if operations == nil {
			serviceUnavailable(writer, request)
			return
		}
		organizations, err := operations.ListOrganizations(request.Context(), principal)
		if err != nil {
			WriteError(writer, http.StatusBadRequest, "organization.list_failed", "organizations could not be listed", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"organizations": organizations})
	}
}

func membershipHandler(authenticator Authenticator, operations OrganizationOperations, action string) http.HandlerFunc {
	type input struct {
		AccountID string `json:"account_id"`
		Role      string `json:"role,omitempty"`
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := requirePrincipal(writer, request, authenticator, false)
		if !ok {
			return
		}
		var body input
		if err := DecodeJSON(writer, request, &body); err != nil {
			return
		}
		if operations == nil || body.AccountID == "" {
			serviceUnavailable(writer, request)
			return
		}
		orgID := request.PathValue("orgID")
		var result MembershipResult
		var err error
		switch action {
		case "add":
			result, err = operations.AddMember(request.Context(), principal, orgID, body.AccountID, body.Role)
		case "remove":
			err = operations.RemoveMember(request.Context(), principal, orgID, body.AccountID)
		case "role-set":
			result, err = operations.SetMemberRole(request.Context(), principal, orgID, body.AccountID, body.Role)
		}
		if err != nil {
			WriteError(writer, http.StatusForbidden, "organization.member_change_failed", "membership could not be changed", request.Header.Get("X-Request-ID"), err)
			return
		}
		if action == "remove" {
			writeJSON(writer, http.StatusOK, map[string]bool{"removed": true})
			return
		}
		writeJSON(writer, http.StatusOK, result)
	}
}

func auditListHandler(authenticator Authenticator, operations AuditOperations) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := requirePrincipal(writer, request, authenticator, false)
		if !ok {
			return
		}
		if operations == nil {
			serviceUnavailable(writer, request)
			return
		}
		events, err := operations.ListAudit(request.Context(), principal)
		if err != nil {
			WriteError(writer, http.StatusForbidden, "audit.list_failed", "audit events could not be listed", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]any{"events": events})
	}
}

func requirePrincipal(writer http.ResponseWriter, request *http.Request, authenticator Authenticator, platformAdmin bool) (Principal, bool) {
	principal, err := authenticateRequest(request.Context(), request, authenticator)
	if err != nil {
		WriteError(writer, http.StatusUnauthorized, "auth.unauthenticated", "authentication is required", request.Header.Get("X-Request-ID"), err)
		return Principal{}, false
	}
	if principal.Account.MustChangePassword {
		WriteError(writer, http.StatusForbidden, "auth.initial_password_change_required", "initial password must be changed", request.Header.Get("X-Request-ID"), errors.New("initial password"))
		return Principal{}, false
	}
	if platformAdmin && !principal.Account.PlatformAdmin {
		WriteError(writer, http.StatusForbidden, "auth.forbidden", "platform administrator role is required", request.Header.Get("X-Request-ID"), errors.New("not platform administrator"))
		return Principal{}, false
	}
	return principal, true
}

func serviceUnavailable(writer http.ResponseWriter, request *http.Request) {
	WriteError(writer, http.StatusServiceUnavailable, "service.unavailable", "service is unavailable", request.Header.Get("X-Request-ID"), errors.New("nil operation service"))
}
