package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/TashanGKD/tashan-compute/internal/audit"
)

type AuditStore struct {
	db *sql.DB
}

func NewAuditStore(db *sql.DB) *AuditStore {
	return &AuditStore{db: db}
}

func (store *AuditStore) Append(ctx context.Context, event audit.Event) (int64, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin audit append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var previousHash []byte
	err = tx.QueryRowContext(ctx, `SELECT event_hash FROM audit_events ORDER BY id DESC LIMIT 1 FOR UPDATE`).Scan(&previousHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("read audit chain head: %w", err)
	}
	metadata, err := json.Marshal(audit.Redact(event.Metadata))
	if err != nil {
		return 0, fmt.Errorf("encode audit metadata: %w", err)
	}
	payload, err := json.Marshal(struct {
		Event        audit.Event     `json:"event"`
		Metadata     json.RawMessage `json:"metadata"`
		PreviousHash []byte          `json:"previous_hash"`
	}{Event: event, Metadata: metadata, PreviousHash: previousHash})
	if err != nil {
		return 0, fmt.Errorf("encode audit hash payload: %w", err)
	}
	eventHash := sha256.Sum256(payload)

	var id int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO audit_events (
			occurred_at, request_id, actor_account_id, actor_device_id, effective_role,
			capability_id, target_type, target_id, outcome, source_ip, user_agent,
			metadata, previous_hash, event_hash
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`,
		event.OccurredAt, event.RequestID, nullable(event.ActorAccountID), nullable(event.ActorDeviceID), nullable(event.EffectiveRole),
		event.CapabilityID, nullable(event.TargetType), nullable(event.TargetID), event.Outcome, nullable(event.SourceIP), nullable(event.UserAgent),
		metadata, nullableBytes(previousHash), eventHash[:],
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert audit event: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit audit append: %w", err)
	}
	return id, nil
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func (store *AuditStore) List(ctx context.Context, limit int) ([]audit.Record, error) {
	if limit < 1 || limit > 1000 {
		return nil, errors.New("audit limit must be between 1 and 1000")
	}
	rows, err := store.db.QueryContext(ctx, `SELECT id, occurred_at, request_id, actor_account_id::text, actor_device_id::text, effective_role, capability_id, target_type, target_id, outcome, source_ip::text, user_agent, metadata FROM audit_events ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []audit.Record
	for rows.Next() {
		var record audit.Record
		var actorAccount, actorDevice, role, targetType, targetID, sourceIP, userAgent sql.NullString
		var metadata []byte
		if err := rows.Scan(&record.ID, &record.OccurredAt, &record.RequestID, &actorAccount, &actorDevice, &role, &record.CapabilityID, &targetType, &targetID, &record.Outcome, &sourceIP, &userAgent, &metadata); err != nil {
			return nil, err
		}
		record.ActorAccountID, record.ActorDeviceID, record.EffectiveRole = actorAccount.String, actorDevice.String, role.String
		record.TargetType, record.TargetID, record.SourceIP, record.UserAgent = targetType.String, targetID.String, sourceIP.String, userAgent.String
		if err := json.Unmarshal(metadata, &record.Metadata); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}
