package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TashanGKD/tashan-compute/capabilities"
	"github.com/TashanGKD/tashan-compute/internal/apicmd"
	"github.com/TashanGKD/tashan-compute/internal/app"
	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/buildinfo"
	"github.com/TashanGKD/tashan-compute/internal/capability"
	"github.com/TashanGKD/tashan-compute/internal/config"
	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/TashanGKD/tashan-compute/internal/secretfile"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cmd := apicmd.NewRoot(apicmd.Dependencies{Build: buildRuntime})
	cmd.SetContext(ctx)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func buildRuntime(ctx context.Context) (apicmd.Runtime, error) {
	cfg, err := config.Load(config.EnvSource{})
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = db.Close() }
	if err := db.PingContext(ctx); err != nil {
		cleanup()
		return nil, err
	}
	if err := store.Migrate(ctx, db); err != nil {
		cleanup()
		return nil, err
	}
	privateBytes, err := secretfile.ReadBase64(cfg.AccessPrivateKeyFile, ed25519.PrivateKeySize)
	if err != nil {
		cleanup()
		return nil, err
	}
	publicBytes, err := secretfile.ReadBase64(cfg.AccessPublicKeyFile, ed25519.PublicKeySize)
	if err != nil {
		cleanup()
		return nil, err
	}
	pepper, err := secretfile.ReadBase64(cfg.RefreshPepperFile, 32)
	if err != nil {
		cleanup()
		return nil, err
	}
	manifestFile, err := capabilities.Files.Open("manifest.json")
	if err != nil {
		cleanup()
		return nil, err
	}
	manifest, err := capability.Load(manifestFile)
	_ = manifestFile.Close()
	if err != nil {
		cleanup()
		return nil, err
	}

	now := time.Now
	hasher := auth.NewPasswordHasher(auth.DefaultPasswordParams())
	accounts := store.NewAccountStore(db)
	devices := store.NewDeviceStore(db)
	sessions := store.NewSessionStore(db)
	authService := auth.NewService(accounts, hasher)
	organizationOperations := app.NewOrganizationOperations(store.NewOrganizationStore(db))
	platformOperations := app.NewPlatformOperations(accounts, hasher, organizationOperations, now)
	authOperations := app.NewAuthOperations(app.AuthDependencies{
		Accounts: accounts, Devices: devices, Sessions: sessions, AuthService: authService,
		SessionManager: auth.NewSessionManager(sessions, pepper, 30*24*time.Hour, now, rand.Reader),
		AccessSigner:   auth.NewAccessTokenSigner(ed25519.PrivateKey(privateBytes), "tashan-compute", "tcompute", 10*time.Minute, now),
		AccessLifetime: 10 * time.Minute, Hasher: hasher, Now: now,
	})
	authenticator := httpapi.NewTokenAuthenticator(
		auth.NewAccessTokenVerifier(ed25519.PublicKey(publicBytes), "tashan-compute", "tcompute", now),
		store.NewPrincipalStore(db),
	)
	handler := httpapi.NewServer(httpapi.ServerOptions{
		Version: buildinfo.Version, Capabilities: manifest, Authenticator: authenticator,
		AuthOperations: authOperations, AdminUsers: app.NewAdminUsers(authService),
		DeviceOperations: app.NewDeviceOperations(devices, now), PlatformOperations: platformOperations,
		OrganizationOperations: organizationOperations, AuditOperations: app.NewAuditOperations(store.NewAuditStore(db)),
	})
	server := &http.Server{
		Addr: cfg.ListenAddress, Handler: handler,
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second,
		WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second,
	}
	return app.NewAPIRuntime(server, cleanup), nil
}
