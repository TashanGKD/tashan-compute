package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/admin"
	"github.com/TashanGKD/tashan-compute/internal/auth"
	store "github.com/TashanGKD/tashan-compute/internal/store/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/term"
)

func main() {
	cmd := admin.NewRoot(admin.Dependencies{
		RepositoryFactory: openBootstrapRepository,
		Hasher:            auth.NewPasswordHasher(auth.DefaultPasswordParams()),
		IsTerminal: func() bool {
			return term.IsTerminal(int(os.Stdin.Fd()))
		},
		ReadPassword: func(prompt string) (string, error) {
			fmt.Fprint(os.Stderr, prompt)
			value, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			return string(value), err
		},
		Stdin: os.Stdin,
	})
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func openBootstrapRepository(ctx context.Context) (admin.BootstrapRepository, func(), error) {
	databaseURL := os.Getenv("TCOMPUTE_DATABASE_URL")
	if databaseURL == "" {
		return nil, nil, errors.New("TCOMPUTE_DATABASE_URL is required for server-local bootstrap")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("open bootstrap database: %w", err)
	}
	cleanup := func() { _ = db.Close() }
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("connect bootstrap database: %w", err)
	}
	if err := store.Migrate(ctx, db); err != nil {
		cleanup()
		return nil, nil, err
	}
	return store.NewAccountStore(db), cleanup, nil
}
