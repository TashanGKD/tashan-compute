package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

type AccountStore struct {
	db *sql.DB
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
