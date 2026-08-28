package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type LoginInput struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	DeviceLabel       string `json:"device_label"`
	DeviceFingerprint string `json:"device_fingerprint"`
}

type LoginResult struct {
	AccessToken        string    `json:"access_token"`
	RefreshToken       string    `json:"refresh_token"`
	AccessExpiresAt    time.Time `json:"access_expires_at"`
	MustChangePassword bool      `json:"must_change_password"`
}

type PasswordChangeInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type PasswordChangeResult struct {
	SessionsRevoked int64 `json:"sessions_revoked"`
	LoginRequired   bool  `json:"login_required"`
}

type AuthOperations interface {
	Login(context.Context, LoginInput) (LoginResult, error)
	InitialPasswordChange(context.Context, Principal, PasswordChangeInput) (PasswordChangeResult, error)
	PasswordChange(context.Context, Principal, PasswordChangeInput) (PasswordChangeResult, error)
	Refresh(context.Context, string) (LoginResult, error)
	Logout(context.Context, Principal) error
}

func loginHandler(operations AuthOperations) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if operations == nil {
			WriteError(writer, http.StatusServiceUnavailable, "service.unavailable", "authentication service is unavailable", request.Header.Get("X-Request-ID"), errors.New("nil auth operations"))
			return
		}
		var input LoginInput
		if err := DecodeJSON(writer, request, &input); err != nil {
			return
		}
		if input.Username == "" || input.Password == "" || input.DeviceLabel == "" || input.DeviceFingerprint == "" {
			WriteError(writer, http.StatusBadRequest, "request.invalid", "username, password, device_label and device_fingerprint are required", request.Header.Get("X-Request-ID"), errors.New("missing login field"))
			return
		}
		result, err := operations.Login(request.Context(), input)
		if err != nil {
			WriteError(writer, http.StatusUnauthorized, "auth.invalid_credentials", "username or password is invalid", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	}
}

func passwordChangeHandler(authenticator Authenticator, operations AuthOperations, initial bool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, err := authenticateRequest(request.Context(), request, authenticator)
		if err != nil {
			WriteError(writer, http.StatusUnauthorized, "auth.unauthenticated", "authentication is required", request.Header.Get("X-Request-ID"), err)
			return
		}
		if initial != principal.Account.MustChangePassword {
			WriteError(writer, http.StatusForbidden, "auth.password_state_invalid", "password change route does not match account state", request.Header.Get("X-Request-ID"), errors.New("password state mismatch"))
			return
		}
		if operations == nil {
			WriteError(writer, http.StatusServiceUnavailable, "service.unavailable", "authentication service is unavailable", request.Header.Get("X-Request-ID"), errors.New("nil auth operations"))
			return
		}
		var input PasswordChangeInput
		if err := DecodeJSON(writer, request, &input); err != nil {
			return
		}
		if input.CurrentPassword == "" || input.NewPassword == "" {
			WriteError(writer, http.StatusBadRequest, "request.invalid", "current_password and new_password are required", request.Header.Get("X-Request-ID"), errors.New("missing password field"))
			return
		}
		var result PasswordChangeResult
		if initial {
			result, err = operations.InitialPasswordChange(request.Context(), principal, input)
		} else {
			result, err = operations.PasswordChange(request.Context(), principal, input)
		}
		if err != nil {
			WriteError(writer, http.StatusBadRequest, "auth.password_change_failed", "password could not be changed", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	}
}

func refreshHandler(operations AuthOperations) http.HandlerFunc {
	type refreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}
	return func(writer http.ResponseWriter, request *http.Request) {
		var input refreshRequest
		if err := DecodeJSON(writer, request, &input); err != nil {
			return
		}
		if operations == nil || input.RefreshToken == "" {
			WriteError(writer, http.StatusUnauthorized, "auth.refresh_invalid", "refresh token is invalid", request.Header.Get("X-Request-ID"), errors.New("invalid refresh"))
			return
		}
		result, err := operations.Refresh(request.Context(), input.RefreshToken)
		if err != nil {
			WriteError(writer, http.StatusUnauthorized, "auth.refresh_invalid", "refresh token is invalid", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, result)
	}
}

func logoutHandler(authenticator Authenticator, operations AuthOperations) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		principal, err := authenticateRequest(request.Context(), request, authenticator)
		if err != nil {
			WriteError(writer, http.StatusUnauthorized, "auth.unauthenticated", "authentication is required", request.Header.Get("X-Request-ID"), err)
			return
		}
		if operations == nil {
			WriteError(writer, http.StatusServiceUnavailable, "service.unavailable", "authentication service is unavailable", request.Header.Get("X-Request-ID"), errors.New("nil auth operations"))
			return
		}
		if err := operations.Logout(request.Context(), principal); err != nil {
			WriteError(writer, http.StatusBadRequest, "auth.logout_failed", "session could not be revoked", request.Header.Get("X-Request-ID"), err)
			return
		}
		writeJSON(writer, http.StatusOK, map[string]bool{"revoked": true})
	}
}

func whoamiHandler(authenticator Authenticator) http.HandlerFunc {
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
		writeJSON(writer, http.StatusOK, map[string]any{
			"account_id": principal.Account.ID, "username": principal.Account.Username,
			"platform_admin": principal.Account.PlatformAdmin, "device_id": principal.DeviceID,
		})
	}
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
