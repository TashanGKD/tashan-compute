package httpapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/identity"
)

func TestTokenAuthenticatorLoadsCurrentServerPrincipal(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	signer := auth.NewAccessTokenSigner(privateKey, "tashan-compute", "tcompute", 10*time.Minute, func() time.Time { return now })
	verifier := auth.NewAccessTokenVerifier(publicKey, "tashan-compute", "tcompute", func() time.Time { return now })
	token, err := signer.Issue(auth.AccessIdentity{AccountID: "account-1", DeviceID: "device-1", SessionID: "session-1", PasswordVersion: 1})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	loader := &fixturePrincipalLoader{principal: identity.Principal{
		Account:  identity.Account{ID: "account-1", PlatformAdmin: false},
		DeviceID: "device-1", SessionID: "session-1",
	}}
	authenticator := NewTokenAuthenticator(verifier, loader)

	principal, err := authenticator.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if principal.Account.PlatformAdmin {
		t.Fatal("token authenticator invented platform administrator role")
	}
	if loader.received.AccountID != "account-1" || loader.received.DeviceID != "device-1" {
		t.Fatalf("loader received = %+v", loader.received)
	}
}

type fixturePrincipalLoader struct {
	principal identity.Principal
	received  auth.AccessIdentity
}

func (loader *fixturePrincipalLoader) Load(_ context.Context, access auth.AccessIdentity) (identity.Principal, error) {
	loader.received = access
	return loader.principal, nil
}
