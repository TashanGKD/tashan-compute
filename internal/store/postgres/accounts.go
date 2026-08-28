package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

type AccountStore struct {
	db *sql.DB
}

func (store *AccountStore) ResetPasswordAndRevokeSessions(ctx context.Context, accountID, passwordHash string, now time.Time) (int64, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin password reset: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET password_hash = $1, password_version = password_version + 1,
		    must_change_password = true, updated_at = $2
		WHERE id = $3`, passwordHash, now, accountID)
	if err != nil {
		return 0, fmt.Errorf("reset account password: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count reset accounts: %w", err)
	}
	if updated != 1 {
		return 0, identity.ErrAccountNotFound
	}
	result, err = tx.ExecContext(ctx, `
		UPDATE sessions SET revoked_at = COALESCE(revoked_at, $1)
		WHERE account_id = $2 AND revoked_at IS NULL`, now, accountID)
	if err != nil {
		return 0, fmt.Errorf("revoke account sessions: %w", err)
	}
	revoked, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count revoked sessions: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit password reset: %w", err)
	}
	return revoked, nil
}

func (store *AccountStore) FindByUsername(ctx context.Context, username string) (identity.Account, error) {
	var account identity.Account
	err := store.db.QueryRowContext(ctx, `
		SELECT id, username, password_hash, platform_admin, password_version, must_change_password, disabled_at
		FROM accounts
		WHERE username = $1`, username).Scan(
		&account.ID,
		&account.Username,
		&account.PasswordHash,
		&account.PlatformAdmin,
		&account.PasswordVersion,
		&account.MustChangePassword,
		&account.DisabledAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return identity.Account{}, identity.ErrAccountNotFound
	}
	if err != nil {
		return identity.Account{}, fmt.Errorf("find account by username: %w", err)
	}
	return account, nil
}

func NewAccountStore(db *sql.DB) *AccountStore {
	return &AccountStore{db: db}
}

func (store *AccountStore) Create(ctx context.Context, account identity.Account) (identity.Account, error) {
	row := store.db.QueryRowContext(ctx, `
		INSERT INTO accounts (username, password_hash, password_version, must_change_password)
		VALUES ($1, $2, $3, $4)
		RETURNING id, platform_admin, disabled_at`,
		account.Username,
		account.PasswordHash,
		account.PasswordVersion,
		account.MustChangePassword,
	)
	if err := row.Scan(&account.ID, &account.PlatformAdmin, &account.DisabledAt); err != nil {
		return identity.Account{}, fmt.Errorf("create account: %w", err)
	}
	return account, nil
}

func (store *AccountStore) Bootstrap(ctx context.Context, account identity.Account) (identity.Account, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return identity.Account{}, fmt.Errorf("begin administrator bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(827436701245194)`); err != nil {
		return identity.Account{}, fmt.Errorf("lock administrator bootstrap: %w", err)
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM accounts WHERE platform_admin)`).Scan(&exists); err != nil {
		return identity.Account{}, fmt.Errorf("check administrator bootstrap: %w", err)
	}
	if exists {
		return identity.Account{}, identity.ErrBootstrapCompleted
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO accounts (username, password_hash, platform_admin, password_version, must_change_password)
		VALUES ($1, $2, true, $3, false)
		RETURNING id, platform_admin, must_change_password, disabled_at`,
		account.Username, account.PasswordHash, account.PasswordVersion,
	).Scan(&account.ID, &account.PlatformAdmin, &account.MustChangePassword, &account.DisabledAt)
	if err != nil {
		return identity.Account{}, fmt.Errorf("create platform administrator: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return identity.Account{}, fmt.Errorf("commit administrator bootstrap: %w", err)
	}
	return account, nil
}
