package app

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type APIRuntime struct {
	server  *http.Server
	cleanup func()
}

func NewAPIRuntime(server *http.Server, cleanup func()) *APIRuntime {
	return &APIRuntime{server: server, cleanup: cleanup}
}

func (runtime *APIRuntime) Serve(ctx context.Context) error {
	defer runtime.cleanup()
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = runtime.server.Shutdown(shutdownCtx)
	}()
	err := runtime.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		<-shutdownDone
		return nil
	}
	return err
}
