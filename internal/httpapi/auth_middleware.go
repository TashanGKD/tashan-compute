package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type Principal struct {
	Account   identity.Account
	DeviceID  string
	SessionID string
}

type Authenticator interface {
	Authenticate(context.Context, string) (Principal, error)
}

func authenticateRequest(ctx context.Context, request *http.Request, authenticator Authenticator) (Principal, error) {
	if authenticator == nil {
		return Principal{}, ErrUnauthenticated
	}
	header := request.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") || strings.Count(header, " ") != 1 {
		return Principal{}, ErrUnauthenticated
	}
	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" {
		return Principal{}, ErrUnauthenticated
	}
	principal, err := authenticator.Authenticate(ctx, token)
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}
	return principal, nil
}
