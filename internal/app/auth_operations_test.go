package app_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/app"
	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/identity"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	"github.com/TashanGKD/tashan-compute/internal/testkit"
)

func TestManagedUserInitialPasswordLifecycleRevokesOldSession(t *testing.T) {
	ctx := context.Background()
	db := testkit.Postgres(t)
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := store.Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	hasher := auth.NewPasswordHasher(auth.DefaultPasswordParams())
	adminHash, _ := hasher.Hash("platform admin fixture password", "root")
	accounts := store.NewAccountStore(db)
	adminAccount, err := accounts.Bootstrap(ctx, identity.Account{Username: "root", PasswordHash: adminHash, PasswordVersion: 1})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	authService := auth.NewService(accounts, hasher)
	if _, err := authService.CreateAccount(ctx, identity.Actor{AccountID: adminAccount.ID, PlatformAdmin: true}, auth.CreateAccountInput{
		Username: "alice", InitialPassword: "alice initial fixture password",
	}); err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Now
	sessions := store.NewSessionStore(db)
	operations := app.NewAuthOperations(app.AuthDependencies{
		Accounts: accounts, Devices: store.NewDeviceStore(db), Sessions: sessions,
		AuthService:    authService,
		SessionManager: auth.NewSessionManager(sessions, []byte("fixture refresh pepper at least 32 bytes"), 30*24*time.Hour, now, rand.Reader),
		AccessSigner:   auth.NewAccessTokenSigner(privateKey, "tashan-compute", "tcompute", 10*time.Minute, now),
		AccessLifetime: 10 * time.Minute, Hasher: hasher, Now: now,
	})

	login, err := operations.Login(ctx, httpapi.LoginInput{
		Username: "alice", Password: "alice initial fixture password", DeviceLabel: "laptop", DeviceFingerprint: "device-fingerprint",
	})
	if err != nil {
		t.Fatalf("Login(initial) error = %v", err)
	}
	if !login.MustChangePassword {
		t.Fatal("initial login did not require password change")
	}
	verifier := auth.NewAccessTokenVerifier(publicKey, "tashan-compute", "tcompute", now)
	access, err := verifier.Verify(login.AccessToken)
	if err != nil {
		t.Fatalf("Verify(initial) error = %v", err)
	}
	principal, err := store.NewPrincipalStore(db).Load(ctx, access)
	if err != nil {
		t.Fatalf("Load(initial principal) error = %v", err)
	}
	result, err := operations.InitialPasswordChange(ctx, principal, httpapi.PasswordChangeInput{
		CurrentPassword: "alice initial fixture password", NewPassword: "alice permanent fixture password",
	})
	if err != nil {
		t.Fatalf("InitialPasswordChange() error = %v", err)
	}
	if !result.LoginRequired || result.SessionsRevoked != 1 {
		t.Fatalf("change result = %+v", result)
	}
	if _, err := store.NewPrincipalStore(db).Load(ctx, access); err == nil {
		t.Fatal("old access session survived password change")
	}
	secondLogin, err := operations.Login(ctx, httpapi.LoginInput{
		Username: "alice", Password: "alice permanent fixture password", DeviceLabel: "laptop", DeviceFingerprint: "device-fingerprint",
	})
	if err != nil || secondLogin.MustChangePassword {
		t.Fatalf("Login(permanent) = %+v, %v", secondLogin, err)
	}
	refreshed, err := operations.Refresh(ctx, secondLogin.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	refreshedAccess, err := verifier.Verify(refreshed.AccessToken)
	if err != nil {
		t.Fatalf("Verify(refreshed) error = %v", err)
	}
	refreshedPrincipal, err := store.NewPrincipalStore(db).Load(ctx, refreshedAccess)
	if err != nil {
		t.Fatalf("Load(refreshed principal) error = %v", err)
	}
	if err := operations.Logout(ctx, refreshedPrincipal); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := store.NewPrincipalStore(db).Load(ctx, refreshedAccess); err == nil {
		t.Fatal("logged-out access session remained valid")
	}
}
