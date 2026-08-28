package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

func TestLoginContractRejectsClientRoleAndAcceptsManagedCredentials(t *testing.T) {
	operations := &fixtureAuthOperations{}
	handler := NewServer(ServerOptions{Version: "test", AuthOperations: operations})

	bad := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"username":"alice","password":"fixture-password","device_label":"mac","device_fingerprint":"fp","role":"platform_admin"}`))
	badRecorder := httptest.NewRecorder()
	handler.ServeHTTP(badRecorder, bad)
	if badRecorder.Code != http.StatusBadRequest {
		t.Fatalf("role injection status = %d body=%s", badRecorder.Code, badRecorder.Body.String())
	}
	if operations.loginCalls != 0 {
		t.Fatal("role injection reached auth service")
	}

	good := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"username":"alice","password":"fixture-password","device_label":"mac","device_fingerprint":"fp"}`))
	goodRecorder := httptest.NewRecorder()
	handler.ServeHTTP(goodRecorder, good)
	if goodRecorder.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", goodRecorder.Code, goodRecorder.Body.String())
	}
	if operations.loginCalls != 1 || operations.username != "alice" {
		t.Fatalf("operations = %+v", operations)
	}
	if !strings.Contains(goodRecorder.Body.String(), `"must_change_password":true`) {
		t.Fatalf("login response = %s", goodRecorder.Body.String())
	}
}

func TestInitialPasswordSessionCanOnlyChangePasswordOrLogout(t *testing.T) {
	authenticator := &fixtureAuthenticator{principals: map[string]Principal{
		"initial": {Account: identity.Account{ID: "account-1", MustChangePassword: true}},
	}}
	operations := &fixtureAuthOperations{}
	handler := NewServer(ServerOptions{Version: "test", Authenticator: authenticator, AuthOperations: operations})

	whoami := httptest.NewRequest(http.MethodGet, "/v1/auth/whoami", nil)
	whoami.Header.Set("Authorization", "Bearer initial")
	whoamiRecorder := httptest.NewRecorder()
	handler.ServeHTTP(whoamiRecorder, whoami)
	if whoamiRecorder.Code != http.StatusForbidden {
		t.Fatalf("whoami status = %d", whoamiRecorder.Code)
	}

	change := httptest.NewRequest(http.MethodPost, "/v1/auth/password/initial-change", strings.NewReader(`{"current_password":"fixture-password","new_password":"new-long-fixture-password"}`))
	change.Header.Set("Authorization", "Bearer initial")
	changeRecorder := httptest.NewRecorder()
	handler.ServeHTTP(changeRecorder, change)
	if changeRecorder.Code != http.StatusOK {
		t.Fatalf("initial change status = %d body=%s", changeRecorder.Code, changeRecorder.Body.String())
	}
}

type fixtureAuthOperations struct {
	loginCalls int
	username   string
}

func (operations *fixtureAuthOperations) Login(_ context.Context, input LoginInput) (LoginResult, error) {
	operations.loginCalls++
	operations.username = input.Username
	return LoginResult{
		AccessToken: "fixture-access", RefreshToken: "fixture-refresh",
		AccessExpiresAt: time.Now().Add(time.Minute), MustChangePassword: true,
	}, nil
}

func (operations *fixtureAuthOperations) InitialPasswordChange(context.Context, Principal, PasswordChangeInput) (PasswordChangeResult, error) {
	return PasswordChangeResult{SessionsRevoked: 1, LoginRequired: true}, nil
}

func (operations *fixtureAuthOperations) PasswordChange(context.Context, Principal, PasswordChangeInput) (PasswordChangeResult, error) {
	return PasswordChangeResult{SessionsRevoked: 1, LoginRequired: true}, nil
}

func (operations *fixtureAuthOperations) Refresh(context.Context, string) (LoginResult, error) {
	return LoginResult{}, nil
}

func (operations *fixtureAuthOperations) Logout(context.Context, Principal) error { return nil }
