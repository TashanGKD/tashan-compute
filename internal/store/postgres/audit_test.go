package postgres_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/audit"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
)

func TestAuditStoreRedactsAndPreventsMutation(t *testing.T) {
	db := testkit.Postgres(t)
	resetPublicSchema(t, db)
	if err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	repository := store.NewAuditStore(db)
	event := audit.Event{
		OccurredAt:   time.Now().UTC(),
		RequestID:    "req-1",
		CapabilityID: "admin.user.create",
		Outcome:      "allowed",
		Metadata: map[string]any{
			"username": "alice",
			"password": "fixture-password",
		},
	}
	id, err := repository.Append(context.Background(), event)
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	var metadata string
	if err := db.QueryRow(`SELECT metadata::text FROM audit_events WHERE id = $1`, id).Scan(&metadata); err != nil {
		t.Fatalf("query metadata: %v", err)
	}
	if strings.Contains(metadata, "fixture-password") || !strings.Contains(metadata, "[REDACTED]") {
		t.Fatalf("metadata = %s", metadata)
	}
	if _, err := db.Exec(`UPDATE audit_events SET outcome = 'failed' WHERE id = $1`, id); err == nil {
		t.Fatal("audit event update was accepted")
	}
}
