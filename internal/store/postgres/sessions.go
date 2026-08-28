package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/auth"
)

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (store *SessionStore) Create(ctx context.Context, session auth.Session) error {
	_, err := store.db.ExecContext(ctx, `
		INSERT INTO sessions (id, account_id, device_id, refresh_token_hash, rotation_family, password_version, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		session.ID, session.AccountID, session.DeviceID, session.RefreshTokenHash,
		session.FamilyID, session.PasswordVersion, session.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (store *SessionStore) FindByRefreshHash(ctx context.Context, hash []byte) (auth.Session, error) {
	return scanSession(store.db.QueryRowContext(ctx, sessionSelect+` WHERE refresh_token_hash = $1`, hash))
}

func (store *SessionStore) Rotate(ctx context.Context, currentHash []byte, next auth.Session, now time.Time) (auth.Session, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return auth.Session{}, fmt.Errorf("begin session rotation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := scanSession(tx.QueryRowContext(ctx, sessionSelect+` WHERE refresh_token_hash = $1 FOR UPDATE`, currentHash))
	if err != nil {
		return auth.Session{}, err
	}
	if current.RevokedAt != nil {
		return current, auth.ErrSessionRevoked
	}
	if current.RotatedAt != nil {
		return current, auth.ErrRefreshReplay
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sessions SET rotated_at = $1 WHERE id = $2`, now, current.ID); err != nil {
		return auth.Session{}, fmt.Errorf("mark session rotated: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO sessions (id, account_id, device_id, refresh_token_hash, rotation_family, password_version, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		next.ID, next.AccountID, next.DeviceID, next.RefreshTokenHash,
		next.FamilyID, next.PasswordVersion, next.ExpiresAt,
	); err != nil {
		return auth.Session{}, fmt.Errorf("insert rotated session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return auth.Session{}, fmt.Errorf("commit session rotation: %w", err)
	}
	return current, nil
}

func (store *SessionStore) RevokeFamily(ctx context.Context, familyID string, now time.Time) error {
	if _, err := store.db.ExecContext(ctx, `UPDATE sessions SET revoked_at = COALESCE(revoked_at, $1) WHERE rotation_family = $2`, now, familyID); err != nil {
		return fmt.Errorf("revoke session family: %w", err)
	}
	return nil
}

const sessionSelect = `
	SELECT id, account_id, device_id, refresh_token_hash, rotation_family,
	       password_version, expires_at, rotated_at, revoked_at
	FROM sessions`

type rowScanner interface {
	Scan(...any) error
}

func scanSession(row rowScanner) (auth.Session, error) {
	var session auth.Session
	err := row.Scan(
		&session.ID, &session.AccountID, &session.DeviceID, &session.RefreshTokenHash,
		&session.FamilyID, &session.PasswordVersion, &session.ExpiresAt,
		&session.RotatedAt, &session.RevokedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	if err != nil {
		return auth.Session{}, fmt.Errorf("scan session: %w", err)
	}
	return session, nil
}
