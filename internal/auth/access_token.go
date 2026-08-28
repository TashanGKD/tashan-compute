package auth

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidAccessToken = errors.New("invalid access token")

type AccessIdentity struct {
	AccountID       string
	SessionID       string
	DeviceID        string
	PasswordVersion int
}

type accessClaims struct {
	SessionID       string `json:"sid"`
	DeviceID        string `json:"did"`
	PasswordVersion int    `json:"password_version"`
	jwt.RegisteredClaims
}

type AccessTokenSigner struct {
	privateKey ed25519.PrivateKey
	issuer     string
	audience   string
	lifetime   time.Duration
	now        func() time.Time
}

func NewAccessTokenSigner(privateKey ed25519.PrivateKey, issuer, audience string, lifetime time.Duration, now func() time.Time) AccessTokenSigner {
	return AccessTokenSigner{privateKey: privateKey, issuer: issuer, audience: audience, lifetime: lifetime, now: now}
}

func (signer AccessTokenSigner) Issue(identity AccessIdentity) (string, error) {
	if len(signer.privateKey) != ed25519.PrivateKeySize || identity.AccountID == "" || identity.SessionID == "" || identity.DeviceID == "" || identity.PasswordVersion < 1 {
		return "", ErrInvalidAccessToken
	}
	now := signer.now().UTC()
	claims := accessClaims{
		SessionID:       identity.SessionID,
		DeviceID:        identity.DeviceID,
		PasswordVersion: identity.PasswordVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    signer.issuer,
			Subject:   identity.AccountID,
			Audience:  jwt.ClaimStrings{signer.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(signer.lifetime)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(signer.privateKey)
}

type AccessTokenVerifier struct {
	publicKey ed25519.PublicKey
	issuer    string
	audience  string
	now       func() time.Time
}

func NewAccessTokenVerifier(publicKey ed25519.PublicKey, issuer, audience string, now func() time.Time) AccessTokenVerifier {
	return AccessTokenVerifier{publicKey: publicKey, issuer: issuer, audience: audience, now: now}
}

func (verifier AccessTokenVerifier) Verify(raw string) (AccessIdentity, error) {
	if len(verifier.publicKey) != ed25519.PublicKeySize || raw == "" {
		return AccessIdentity{}, ErrInvalidAccessToken
	}
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
			return nil, ErrInvalidAccessToken
		}
		return verifier.publicKey, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(verifier.issuer),
		jwt.WithAudience(verifier.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(verifier.now),
	)
	if err != nil || !token.Valid {
		return AccessIdentity{}, fmt.Errorf("%w", ErrInvalidAccessToken)
	}
	identity := AccessIdentity{
		AccountID:       claims.Subject,
		SessionID:       claims.SessionID,
		DeviceID:        claims.DeviceID,
		PasswordVersion: claims.PasswordVersion,
	}
	if identity.AccountID == "" || identity.SessionID == "" || identity.DeviceID == "" || identity.PasswordVersion < 1 {
		return AccessIdentity{}, ErrInvalidAccessToken
	}
	return identity, nil
}
