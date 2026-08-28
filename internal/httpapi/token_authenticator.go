package httpapi

import (
	"context"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/identity"
)

type PrincipalLoader interface {
	Load(context.Context, auth.AccessIdentity) (identity.Principal, error)
}

type TokenAuthenticator struct {
	verifier auth.AccessTokenVerifier
	loader   PrincipalLoader
}

func NewTokenAuthenticator(verifier auth.AccessTokenVerifier, loader PrincipalLoader) *TokenAuthenticator {
	return &TokenAuthenticator{verifier: verifier, loader: loader}
}

func (authenticator *TokenAuthenticator) Authenticate(ctx context.Context, raw string) (Principal, error) {
	access, err := authenticator.verifier.Verify(raw)
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}
	principal, err := authenticator.loader.Load(ctx, access)
	if err != nil {
		return Principal{}, ErrUnauthenticated
	}
	return principal, nil
}
