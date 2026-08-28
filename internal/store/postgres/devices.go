package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

type DeviceStore struct {
	db *sql.DB
}

func NewDeviceStore(db *sql.DB) *DeviceStore {
	return &DeviceStore{db: db}
}

func (store *DeviceStore) UpsertAfterPasswordLogin(ctx context.Context, accountID, label string, fingerprintHash []byte, now time.Time) (identity.Device, error) {
	var device identity.Device
	err := store.db.QueryRowContext(ctx, `
		INSERT INTO devices (account_id, label, fingerprint_hash, first_seen_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (account_id, fingerprint_hash) DO UPDATE
		SET label = EXCLUDED.label, last_seen_at = EXCLUDED.last_seen_at, revoked_at = NULL
		RETURNING id, account_id, label, first_seen_at, last_seen_at, revoked_at`,
		accountID, label, fingerprintHash, now,
	).Scan(&device.ID, &device.AccountID, &device.Label, &device.FirstSeenAt, &device.LastSeenAt, &device.RevokedAt)
	if err != nil {
		return identity.Device{}, fmt.Errorf("upsert device: %w", err)
	}
	return device, nil
}

func (store *DeviceStore) Revoke(ctx context.Context, accountID, deviceID string, now time.Time) error {
	result, err := store.db.ExecContext(ctx, `UPDATE devices SET revoked_at = COALESCE(revoked_at, $1) WHERE id = $2 AND account_id = $3`, now, deviceID, accountID)
	if err != nil {
		return fmt.Errorf("revoke device: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count revoked devices: %w", err)
	}
	if count != 1 {
		return identity.ErrDeviceNotFound
	}
	return nil
}

func (store *DeviceStore) List(ctx context.Context, accountID string) ([]identity.Device, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT id, account_id, label, first_seen_at, last_seen_at, revoked_at FROM devices WHERE account_id = $1 ORDER BY first_seen_at, id`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()
	var devices []identity.Device
	for rows.Next() {
		var device identity.Device
		if err := rows.Scan(&device.ID, &device.AccountID, &device.Label, &device.FirstSeenAt, &device.LastSeenAt, &device.RevokedAt); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, device)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return devices, nil
}
