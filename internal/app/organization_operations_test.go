package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TashanGKD/tashan-compute/internal/app"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
)

func TestOrganizationAdminManagesExistingUsersWithinOrganization(t *testing.T) {
	ctx := context.Background()
	db := testkit.Postgres(t)
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := store.Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	var adminID, memberID string
	if err := db.QueryRow(`INSERT INTO accounts (username, password_hash, platform_admin, must_change_password) VALUES ('root', 'hash', true, false) RETURNING id`).Scan(&adminID); err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	if err := db.QueryRow(`INSERT INTO accounts (username, password_hash, must_change_password) VALUES ('alice', 'hash', false) RETURNING id`).Scan(&memberID); err != nil {
		t.Fatalf("insert member: %v", err)
	}
	organizations := app.NewOrganizationOperations(store.NewOrganizationStore(db))
	admin := identity.Principal{Account: identity.Account{ID: adminID, PlatformAdmin: true}}
	created, err := organizations.CreateOrganization(ctx, admin, "research-team")
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	membership, err := organizations.AddMember(ctx, admin, created.OrganizationID, memberID, "developer")
	if err != nil {
		t.Fatalf("AddMember() error = %v", err)
	}
	if membership.Role != "developer" {
		t.Fatalf("membership = %+v", membership)
	}
	viewer := identity.Principal{Account: identity.Account{ID: memberID}}
	if _, err := organizations.AddMember(ctx, viewer, created.OrganizationID, adminID, "viewer"); !errors.Is(err, app.ErrOrganizationAdminRequired) {
		t.Fatalf("viewer AddMember() error = %v", err)
	}
	listed, err := organizations.ListOrganizations(ctx, viewer)
	if err != nil || len(listed) != 1 {
		t.Fatalf("ListOrganizations() = %+v, %v", listed, err)
	}
}
