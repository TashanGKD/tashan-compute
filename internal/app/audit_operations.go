package app

import (
	"context"
	"errors"

	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
)

type AuditOperations struct{ audit *store.AuditStore }

func NewAuditOperations(auditStore *store.AuditStore) *AuditOperations {
	return &AuditOperations{audit: auditStore}
}

func (operations *AuditOperations) ListAudit(ctx context.Context, principal identity.Principal) ([]map[string]any, error) {
	if !principal.Account.PlatformAdmin {
		return nil, errors.New("audit access is not authorized")
	}
	records, err := operations.audit.List(ctx, 100)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		result = append(result, map[string]any{
			"id": record.ID, "occurred_at": record.OccurredAt, "request_id": record.RequestID,
			"capability_id": record.CapabilityID, "outcome": record.Outcome,
			"actor_account_id": record.ActorAccountID, "actor_device_id": record.ActorDeviceID,
			"target_type": record.TargetType, "target_id": record.TargetID, "metadata": record.Metadata,
		})
	}
	return result, nil
}

var _ httpapi.AuditOperations = (*AuditOperations)(nil)
