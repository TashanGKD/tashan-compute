package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestRefreshTokenRotatesAndReplayRevokesFamily(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	randomBytes := append(bytes.Repeat([]byte{0x11}, 32), bytes.Repeat([]byte{0x22}, 32)...)
	repository := newMemorySessions()
	manager := NewSessionManager(repository, []byte("fixture pepper with at least 32 bytes"), 30*24*time.Hour, func() time.Time { return now }, bytes.NewReader(randomBytes))

	first, err := manager.Issue(context.Background(), "account-1", "device-1", 1)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	second, err := manager.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if first.RefreshToken == second.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if second.AccountID != "account-1" || second.DeviceID != "device-1" || second.PasswordVersion != 1 {
		t.Fatalf("rotated credential identity = %+v", second)
	}

	if _, err := manager.Refresh(context.Background(), first.RefreshToken); !errors.Is(err, ErrRefreshReplay) {
		t.Fatalf("Refresh(replay) error = %v", err)
	}
	if _, err := manager.Refresh(context.Background(), second.RefreshToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("Refresh(after family revoke) error = %v", err)
	}
}

func TestSessionManagerRejectsShortPepperAndTruncatedRandomness(t *testing.T) {
	repository := newMemorySessions()
	if _, err := NewSessionManager(repository, []byte("short"), time.Hour, time.Now, bytes.NewReader(nil)).Issue(context.Background(), "account", "device", 1); err == nil {
		t.Fatal("Issue() accepted short pepper")
	}
	manager := NewSessionManager(repository, bytes.Repeat([]byte{0x01}, 32), time.Hour, time.Now, bytes.NewReader([]byte{0x01}))
	if _, err := manager.Issue(context.Background(), "account", "device", 1); err == nil {
		t.Fatal("Issue() accepted truncated randomness")
	}
}

type memorySessions struct {
	byHash map[string]Session
}

func newMemorySessions() *memorySessions {
	return &memorySessions{byHash: make(map[string]Session)}
}

func (repository *memorySessions) Create(_ context.Context, session Session) error {
	repository.byHash[string(session.RefreshTokenHash)] = session
	return nil
}

func (repository *memorySessions) FindByRefreshHash(_ context.Context, hash []byte) (Session, error) {
	session, ok := repository.byHash[string(hash)]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (repository *memorySessions) Rotate(_ context.Context, currentHash []byte, next Session, now time.Time) (Session, error) {
	current, ok := repository.byHash[string(currentHash)]
	if !ok {
		return Session{}, ErrSessionNotFound
	}
	if current.RevokedAt != nil {
		return current, ErrSessionRevoked
	}
	if current.RotatedAt != nil {
		return current, ErrRefreshReplay
	}
	current.RotatedAt = &now
	repository.byHash[string(currentHash)] = current
	repository.byHash[string(next.RefreshTokenHash)] = next
	return current, nil
}

func (repository *memorySessions) RevokeFamily(_ context.Context, familyID string, now time.Time) error {
	for hash, session := range repository.byHash {
		if session.FamilyID == familyID {
			session.RevokedAt = &now
			repository.byHash[hash] = session
		}
	}
	return nil
}
