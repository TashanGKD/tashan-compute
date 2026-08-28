package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFoundationRoutesAreRegistered(t *testing.T) {
	handler := NewServer(ServerOptions{Version: "test"})
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, RouteHealth, ""},
		{http.MethodGet, RouteCapabilities, ""},
		{http.MethodPost, RouteLogin, `{}`},
		{http.MethodPost, RouteInitialPasswordChange, `{}`},
		{http.MethodPost, RoutePasswordChange, `{}`},
		{http.MethodPost, RouteRefresh, `{}`},
		{http.MethodPost, RouteLogout, `{}`},
		{http.MethodGet, RouteWhoAmI, ""},
		{http.MethodGet, RouteDevices, ""},
		{http.MethodDelete, RouteDevices + "/device-1", ""},
		{http.MethodPost, RouteAdminUsers, `{}`},
		{http.MethodPost, RouteAdminUsers + "/account-1/reset-password", `{}`},
		{http.MethodPost, RouteAdminUsers + "/account-1/disable", `{}`},
		{http.MethodPost, RouteAdminOrganizations, `{}`},
		{http.MethodGet, RouteOrganizations, ""},
		{http.MethodPost, RouteOrganizations + "/org-1/members", `{}`},
		{http.MethodDelete, RouteOrganizations + "/org-1/members", `{}`},
		{http.MethodPatch, RouteOrganizations + "/org-1/members", `{}`},
		{http.MethodGet, RouteAudit, ""},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.path, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code == http.StatusNotFound || recorder.Code == http.StatusMethodNotAllowed {
				t.Fatalf("route is not registered: status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}
