package httpapi

const (
	RouteHealth                = "/v1/health"
	RouteCapabilities          = "/v1/capabilities"
	RouteLogin                 = "/v1/auth/login"
	RouteInitialPasswordChange = "/v1/auth/password/initial-change"
	RoutePasswordChange        = "/v1/auth/password/change"
	RouteRefresh               = "/v1/auth/refresh"
	RouteLogout                = "/v1/auth/logout"
	RouteWhoAmI                = "/v1/auth/whoami"
	RouteDevices               = "/v1/devices"
	RouteAdminUsers            = "/v1/admin/users"
	RouteAdminOrganizations    = "/v1/admin/organizations"
	RouteOrganizations         = "/v1/organizations"
	RouteAudit                 = "/v1/audit"
)
