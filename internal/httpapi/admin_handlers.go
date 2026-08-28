package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type AdminUserResult struct {
	AccountID          string `json:"account_id"`
	MustChangePassword bool   `json:"must_change_password"`
}

type AdminUserCreator interface {
	Create(context.Context, Principal, string, string) (AdminUserResult, error)
}

type adminUserCreateRequest struct {
	Username        string `json:"username"`
	InitialPassword string `json:"initial_password"`
}

func adminUserCreateHandler(authenticator Authenticator, creator AdminUserCreator) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, err := authenticateRequest(request.Context(), request, authenticator)
		if err != nil {
			WriteError(writer, http.StatusUnauthorized, "auth.unauthenticated", "authentication is required", request.Header.Get("X-Request-ID"), err)
			return
		}
		if principal.Account.MustChangePassword {
			WriteError(writer, http.StatusForbidden, "auth.initial_password_change_required", "initial password must be changed", request.Header.Get("X-Request-ID"), errors.New("initial password"))
			return
		}
		if !principal.Account.PlatformAdmin {
			WriteError(writer, http.StatusForbidden, "auth.forbidden", "platform administrator role is required", request.Header.Get("X-Request-ID"), errors.New("not platform administrator"))
			return
		}
		var input adminUserCreateRequest
		if err := DecodeJSON(writer, request, &input); err != nil {
			return
		}
		if input.Username == "" || input.InitialPassword == "" {
			WriteError(writer, http.StatusBadRequest, "request.invalid", "username and initial_password are required", request.Header.Get("X-Request-ID"), errors.New("missing field"))
			return
		}
		if creator == nil {
			WriteError(writer, http.StatusServiceUnavailable, "service.unavailable", "administrator service is unavailable", request.Header.Get("X-Request-ID"), errors.New("nil service"))
			return
		}
		result, err := creator.Create(request.Context(), principal, input.Username, input.InitialPassword)
		if err != nil {
			WriteError(writer, http.StatusBadRequest, "admin.user.create_failed", "account could not be created", request.Header.Get("X-Request-ID"), err)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(writer).Encode(result)
	}
}
