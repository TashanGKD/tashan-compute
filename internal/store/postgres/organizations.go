package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

type OrganizationStore struct{ db *sql.DB }

func NewOrganizationStore(db *sql.DB) *OrganizationStore { return &OrganizationStore{db: db} }

func (store *OrganizationStore) Create(ctx context.Context, name, creatorID string) (identity.Organization, error) {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return identity.Organization{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var organization identity.Organization
	if err := tx.QueryRowContext(ctx, `INSERT INTO organizations (name, created_by) VALUES ($1, $2) RETURNING id, name`, name, creatorID).Scan(&organization.ID, &organization.Name); err != nil {
		return identity.Organization{}, fmt.Errorf("create organization: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO memberships (organization_id, account_id, role) VALUES ($1, $2, 'org_admin')`, organization.ID, creatorID); err != nil {
		return identity.Organization{}, fmt.Errorf("create organization administrator: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return identity.Organization{}, err
	}
	organization.Role = "org_admin"
	return organization, nil
}

func (store *OrganizationStore) ListForAccount(ctx context.Context, accountID string) ([]identity.Organization, error) {
	rows, err := store.db.QueryContext(ctx, `SELECT o.id, o.name, m.role FROM memberships m JOIN organizations o ON o.id = m.organization_id WHERE m.account_id = $1 ORDER BY o.name, o.id`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var organizations []identity.Organization
	for rows.Next() {
		var organization identity.Organization
		if err := rows.Scan(&organization.ID, &organization.Name, &organization.Role); err != nil {
			return nil, err
		}
		organizations = append(organizations, organization)
	}
	return organizations, rows.Err()
}

func (store *OrganizationStore) Role(ctx context.Context, organizationID, accountID string) (string, error) {
	var role string
	err := store.db.QueryRowContext(ctx, `SELECT role FROM memberships WHERE organization_id = $1 AND account_id = $2`, organizationID, accountID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", identity.ErrPrincipalInvalid
	}
	return role, err
}

func (store *OrganizationStore) UpsertMember(ctx context.Context, membership identity.Membership) (identity.Membership, error) {
	err := store.db.QueryRowContext(ctx, `INSERT INTO memberships (organization_id, account_id, role) VALUES ($1, $2, $3) ON CONFLICT (organization_id, account_id) DO UPDATE SET role = EXCLUDED.role, updated_at = now() RETURNING role`, membership.OrganizationID, membership.AccountID, membership.Role).Scan(&membership.Role)
	return membership, err
}

func (store *OrganizationStore) RemoveMember(ctx context.Context, organizationID, accountID string) error {
	result, err := store.db.ExecContext(ctx, `DELETE FROM memberships WHERE organization_id = $1 AND account_id = $2`, organizationID, accountID)
	if err != nil {
		return err
	}
	count, _ := result.RowsAffected()
	if count != 1 {
		return identity.ErrPrincipalInvalid
	}
	return nil
}
