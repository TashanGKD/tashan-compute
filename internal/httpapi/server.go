package httpapi

import (
	"encoding/json"
	"net/http"
)

type ServerOptions struct {
	Version        string
	Authenticator  Authenticator
	AdminUsers     AdminUserCreator
	AuthOperations AuthOperations
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
	return mux
}
