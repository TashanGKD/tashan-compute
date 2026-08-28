package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

func TestAdminRoutesRejectEveryClientSideEscalation(t *testing.T) {
	authenticator := &fixtureAuthenticator{principals: map[string]Principal{
		"ordinary": {Account: identity.Account{ID: "account-ordinary"}},
		"initial":  {Account: identity.Account{ID: "account-initial", PlatformAdmin: true, MustChangePassword: true}},
		"admin":    {Account: identity.Account{ID: "account-admin", PlatformAdmin: true}},
	}}
	creator := &fixtureAdminUsers{}
	handler := NewServer(ServerOptions{Version: "test", Authenticator: authenticator, AdminUsers: creator})

	tests := []struct {
		name  string
		token string
		want  int
	}{
		{name: "no token", want: http.StatusUnauthorized},
		{name: "self signed", token: "self-signed", want: http.StatusUnauthorized},
		{name: "ordinary user", token: "ordinary", want: http.StatusForbidden},
		{name: "revoked device", token: "revoked", want: http.StatusUnauthorized},
		{name: "initial password", token: "initial", want: http.StatusForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/admin/users", strings.NewReader(`{"username":"bob","initial_password":"fixture-password-2026"}`))
			request.Header.Set("Content-Type", "application/json")
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != test.want {
				t.Fatalf("status = %d body=%s, want %d", recorder.Code, recorder.Body.String(), test.want)
			}
			if strings.Contains(recorder.Body.String(), "fixture-password") {
				t.Fatalf("response leaked password: %s", recorder.Body.String())
			}
		})
	}
	if creator.calls != 0 {
		t.Fatalf("admin service called %d times for rejected requests", creator.calls)
	}
}

func TestAdminRouteAcceptsServerAuthorizedAdministrator(t *testing.T) {
	authenticator := &fixtureAuthenticator{principals: map[string]Principal{
		"admin": {Account: identity.Account{ID: "account-admin", PlatformAdmin: true}, DeviceID: "device-1", SessionID: "session-1"},
	}}
	creator := &fixtureAdminUsers{}
	handler := NewServer(ServerOptions{Version: "test", Authenticator: authenticator, AdminUsers: creator})
	request := httptest.NewRequest(http.MethodPost, "/v1/admin/users", strings.NewReader(`{"username":"bob","initial_password":"fixture-password-2026"}`))
	request.Header.Set("Authorization", "Bearer admin")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if creator.calls != 1 || creator.username != "bob" {
		t.Fatalf("creator = %+v", creator)
	}
	if strings.Contains(recorder.Body.String(), "fixture-password") {
		t.Fatalf("response leaked password: %s", recorder.Body.String())
	}
}

type fixtureAuthenticator struct {
	principals map[string]Principal
}

func (authenticator *fixtureAuthenticator) Authenticate(_ context.Context, token string) (Principal, error) {
	principal, ok := authenticator.principals[token]
	if !ok {
		return Principal{}, errors.New("invalid fixture token")
	}
	return principal, nil
}

type fixtureAdminUsers struct {
	calls    int
	username string
}

func (creator *fixtureAdminUsers) Create(_ context.Context, _ Principal, username, _ string) (AdminUserResult, error) {
	creator.calls++
	creator.username = username
	return AdminUserResult{AccountID: "account-bob", MustChangePassword: true}, nil
}
