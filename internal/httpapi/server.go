package httpapi

import (
	"encoding/json"
	"net/http"
)

type ServerOptions struct {
	Version string
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
	return mux
}
