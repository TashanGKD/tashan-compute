package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAccessTokenRejectsSelfSignedAndExpiredTokens(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	signer := NewAccessTokenSigner(privateKey, "tashan-compute", "tcompute", 10*time.Minute, func() time.Time { return now })
	verifier := NewAccessTokenVerifier(publicKey, "tashan-compute", "tcompute", func() time.Time { return now })

	valid, err := signer.Issue(AccessIdentity{AccountID: "account-1", SessionID: "session-1", DeviceID: "device-1", PasswordVersion: 1})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	if _, err := verifier.Verify(valid); err != nil {
		t.Fatalf("Verify(valid) error = %v", err)
	}

	_, attackerKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey(attacker) error = %v", err)
	}
	selfSigned := signMapClaims(t, attackerKey, now, map[string]any{"role": "platform_admin"})
	if _, err := verifier.Verify(selfSigned); !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("Verify(self-signed) error = %v", err)
	}

	expiredVerifier := NewAccessTokenVerifier(publicKey, "tashan-compute", "tcompute", func() time.Time { return now.Add(11 * time.Minute) })
	if _, err := expiredVerifier.Verify(valid); !errors.Is(err, ErrInvalidAccessToken) {
		t.Fatalf("Verify(expired) error = %v", err)
	}
}

func TestAccessTokenIgnoresClientRoleClaimEvenWithValidSignature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	verifier := NewAccessTokenVerifier(publicKey, "tashan-compute", "tcompute", func() time.Time { return now })
	token := signMapClaims(t, privateKey, now, map[string]any{"role": "platform_admin"})

	identity, err := verifier.Verify(token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if identity.AccountID != "account-1" || identity.DeviceID != "device-1" || identity.SessionID != "session-1" {
		t.Fatalf("Verify() identity = %+v", identity)
	}
}

func signMapClaims(t *testing.T, privateKey ed25519.PrivateKey, now time.Time, extra map[string]any) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss":              "tashan-compute",
		"aud":              "tcompute",
		"sub":              "account-1",
		"sid":              "session-1",
		"did":              "device-1",
		"password_version": 1,
		"iat":              now.Unix(),
		"nbf":              now.Unix(),
		"exp":              now.Add(10 * time.Minute).Unix(),
	}
	for key, value := range extra {
		claims[key] = value
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(privateKey)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return token
}
