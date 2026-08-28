package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/TashanGKD/tashan-compute/internal/capability"
)

type ServerOptions struct {
	Version                string
	Capabilities           []capability.Capability
	Authenticator          Authenticator
	AdminUsers             AdminUserCreator
	AuthOperations         AuthOperations
	DeviceOperations       DeviceOperations
	PlatformOperations     PlatformOperations
	OrganizationOperations OrganizationOperations
	AuditOperations        AuditOperations
}

func NewServer(options ServerOptions) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		}{Status: "ok", Version: options.Version})
	})
	mux.HandleFunc("POST /v1/admin/users", adminUserCreateHandler(options.Authenticator, options.AdminUsers))
	mux.HandleFunc("POST /v1/auth/login", loginHandler(options.AuthOperations))
	mux.HandleFunc("POST /v1/auth/password/initial-change", passwordChangeHandler(options.Authenticator, options.AuthOperations, true))
	mux.HandleFunc("POST /v1/auth/password/change", passwordChangeHandler(options.Authenticator, options.AuthOperations, false))
	mux.HandleFunc("POST /v1/auth/refresh", refreshHandler(options.AuthOperations))
	mux.HandleFunc("POST /v1/auth/logout", logoutHandler(options.Authenticator, options.AuthOperations))
	mux.HandleFunc("GET /v1/auth/whoami", whoamiHandler(options.Authenticator))
	mux.HandleFunc("GET "+RouteCapabilities, capabilitiesHandler(options.Capabilities))
	mux.HandleFunc("GET "+RouteDevices, devicesHandler(options.Authenticator, options.DeviceOperations))
	mux.HandleFunc("DELETE "+RouteDevices+"/{deviceID}", deviceRevokeHandler(options.Authenticator, options.DeviceOperations))
	mux.HandleFunc("POST "+RouteAdminUsers+"/{accountID}/reset-password", adminResetPasswordHandler(options.Authenticator, options.PlatformOperations))
	mux.HandleFunc("POST "+RouteAdminUsers+"/{accountID}/disable", adminDisableUserHandler(options.Authenticator, options.PlatformOperations))
	mux.HandleFunc("POST "+RouteAdminOrganizations, adminCreateOrganizationHandler(options.Authenticator, options.PlatformOperations))
	mux.HandleFunc("GET "+RouteOrganizations, organizationsHandler(options.Authenticator, options.OrganizationOperations))
	mux.HandleFunc("POST "+RouteOrganizations+"/{orgID}/members", membershipHandler(options.Authenticator, options.OrganizationOperations, "add"))
	mux.HandleFunc("DELETE "+RouteOrganizations+"/{orgID}/members", membershipHandler(options.Authenticator, options.OrganizationOperations, "remove"))
	mux.HandleFunc("PATCH "+RouteOrganizations+"/{orgID}/members", membershipHandler(options.Authenticator, options.OrganizationOperations, "role-set"))
	mux.HandleFunc("GET "+RouteAudit, auditListHandler(options.Authenticator, options.AuditOperations))
	return mux
}
