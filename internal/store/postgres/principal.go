package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/identity"
)

type PrincipalStore struct {
	db *sql.DB
}

func NewPrincipalStore(db *sql.DB) *PrincipalStore {
	return &PrincipalStore{db: db}
}

func (store *PrincipalStore) Load(ctx context.Context, access auth.AccessIdentity) (identity.Principal, error) {
	var principal identity.Principal
	err := store.db.QueryRowContext(ctx, `
		SELECT a.id, a.username, a.password_hash, a.platform_admin,
		       a.password_version, a.must_change_password, a.disabled_at,
		       d.id, s.id
		FROM sessions s
		JOIN devices d ON d.id = s.device_id AND d.account_id = s.account_id
		JOIN accounts a ON a.id = s.account_id
		WHERE s.id = $1 AND s.account_id = $2 AND s.device_id = $3
		  AND s.password_version = $4 AND a.password_version = $4
		  AND s.revoked_at IS NULL AND s.expires_at > now()
		  AND d.revoked_at IS NULL AND a.disabled_at IS NULL`,
		access.SessionID, access.AccountID, access.DeviceID, access.PasswordVersion,
	).Scan(
		&principal.Account.ID,
		&principal.Account.Username,
		&principal.Account.PasswordHash,
		&principal.Account.PlatformAdmin,
		&principal.Account.PasswordVersion,
		&principal.Account.MustChangePassword,
		&principal.Account.DisabledAt,
		&principal.DeviceID,
		&principal.SessionID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.Principal{}, identity.ErrPrincipalInvalid
	}
	if err != nil {
		return identity.Principal{}, fmt.Errorf("load principal: %w", err)
	}
	return principal, nil
}
