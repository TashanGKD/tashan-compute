package httpapi

import (
	"encoding/json"
	"net/http"
)

type ServerOptions struct {
	Version       string
	Authenticator Authenticator
	AdminUsers    AdminUserCreator
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
	return mux
}
