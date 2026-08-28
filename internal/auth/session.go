package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionRevoked  = errors.New("session revoked")
	ErrRefreshReplay   = errors.New("refresh token replay")
)

type Session struct {
	ID               string
	AccountID        string
	DeviceID         string
	RefreshTokenHash []byte
	FamilyID         string
	PasswordVersion  int
	ExpiresAt        time.Time
	RotatedAt        *time.Time
	RevokedAt        *time.Time
}

type SessionCredential struct {
	SessionID    string
	RefreshToken string
	ExpiresAt    time.Time
}

type SessionRepository interface {
	Create(context.Context, Session) error
	FindByRefreshHash(context.Context, []byte) (Session, error)
	Rotate(context.Context, []byte, Session, time.Time) (Session, error)
	RevokeFamily(context.Context, string, time.Time) error
}

type SessionManager struct {
	repository SessionRepository
	pepper     []byte
	lifetime   time.Duration
	now        func() time.Time
	random     io.Reader
}

func NewSessionManager(repository SessionRepository, pepper []byte, lifetime time.Duration, now func() time.Time, random io.Reader) SessionManager {
	return SessionManager{repository: repository, pepper: append([]byte(nil), pepper...), lifetime: lifetime, now: now, random: random}
}

func (manager SessionManager) Issue(ctx context.Context, accountID, deviceID string, passwordVersion int) (SessionCredential, error) {
	if len(manager.pepper) < 32 || accountID == "" || deviceID == "" || passwordVersion < 1 || manager.lifetime <= 0 {
		return SessionCredential{}, errors.New("invalid session configuration")
	}
	raw, hash, err := manager.newRefreshToken()
	if err != nil {
		return SessionCredential{}, err
	}
	now := manager.now().UTC()
	session := Session{
		ID:               uuid.NewString(),
		AccountID:        accountID,
		DeviceID:         deviceID,
		RefreshTokenHash: hash,
		FamilyID:         uuid.NewString(),
		PasswordVersion:  passwordVersion,
		ExpiresAt:        now.Add(manager.lifetime),
	}
	if err := manager.repository.Create(ctx, session); err != nil {
		return SessionCredential{}, err
	}
	return SessionCredential{SessionID: session.ID, RefreshToken: raw, ExpiresAt: session.ExpiresAt}, nil
}

func (manager SessionManager) Refresh(ctx context.Context, raw string) (SessionCredential, error) {
	if len(manager.pepper) < 32 || raw == "" {
		return SessionCredential{}, ErrSessionNotFound
	}
	hash := manager.hashToken(raw)
	current, err := manager.repository.FindByRefreshHash(ctx, hash)
	if err != nil {
		return SessionCredential{}, err
	}
	now := manager.now().UTC()
	if current.RevokedAt != nil || !current.ExpiresAt.After(now) {
		return SessionCredential{}, ErrSessionRevoked
	}
	if current.RotatedAt != nil {
		_ = manager.repository.RevokeFamily(ctx, current.FamilyID, now)
		return SessionCredential{}, ErrRefreshReplay
	}
	nextRaw, nextHash, err := manager.newRefreshToken()
	if err != nil {
		return SessionCredential{}, err
	}
	next := Session{
		ID:               uuid.NewString(),
		AccountID:        current.AccountID,
		DeviceID:         current.DeviceID,
		RefreshTokenHash: nextHash,
		FamilyID:         current.FamilyID,
		PasswordVersion:  current.PasswordVersion,
		ExpiresAt:        now.Add(manager.lifetime),
	}
	if _, err := manager.repository.Rotate(ctx, hash, next, now); err != nil {
		if errors.Is(err, ErrRefreshReplay) {
			_ = manager.repository.RevokeFamily(ctx, current.FamilyID, now)
		}
		return SessionCredential{}, err
	}
	return SessionCredential{SessionID: next.ID, RefreshToken: nextRaw, ExpiresAt: next.ExpiresAt}, nil
}

func (manager SessionManager) newRefreshToken() (string, []byte, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(manager.random, bytes); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(bytes)
	return raw, manager.hashToken(raw), nil
}

func (manager SessionManager) hashToken(raw string) []byte {
	mac := hmac.New(sha256.New, manager.pepper)
	_, _ = mac.Write([]byte(raw))
	return mac.Sum(nil)
}
