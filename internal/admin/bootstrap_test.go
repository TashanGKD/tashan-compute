package admin

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/TashanGKD/tashan-compute/internal/identity"
)

func TestRootHelpDoesNotOpenDatabase(t *testing.T) {
	called := false
	cmd := NewRoot(Dependencies{
		RepositoryFactory: func(context.Context) (BootstrapRepository, func(), error) {
			called = true
			return nil, func() {}, nil
		},
	})
	cmd.SetArgs(nil)
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called {
		t.Fatal("no-argument help opened the database")
	}
}

func TestBootstrapRejectsPasswordFlagAndNonInteractiveInput(t *testing.T) {
	cmd := NewRoot(Dependencies{IsTerminal: func() bool { return false }})
	cmd.SetArgs([]string{"bootstrap", "--username", "root", "--password", "leaked"})
	if err := cmd.Execute(); err == nil || err.Error() != "unknown flag: --password" {
		t.Fatalf("--password error = %v", err)
	}

	cmd = NewRoot(Dependencies{IsTerminal: func() bool { return false }})
	cmd.SetArgs([]string{"bootstrap", "--username", "root"})
	if err := cmd.Execute(); err == nil || err.Error() != "interactive terminal or --password-stdin is required" {
		t.Fatalf("non-interactive error = %v", err)
	}
}

func TestBootstrapReadsPasswordFromStdinAndSucceedsOnlyOnce(t *testing.T) {
	repository := &memoryBootstrap{}
	dependencies := Dependencies{
		Repository: repository,
		Hasher:     auth.NewPasswordHasher(auth.DefaultPasswordParams()),
		IsTerminal: func() bool { return false },
		Stdin:      bytes.NewBufferString("very long bootstrap password 2026\n"),
	}
	cmd := NewRoot(dependencies)
	cmd.SetArgs([]string{"bootstrap", "--username", "root", "--password-stdin"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("first bootstrap error = %v", err)
	}

	dependencies.Stdin = bytes.NewBufferString("another long bootstrap password 2026\n")
	cmd = NewRoot(dependencies)
	cmd.SetArgs([]string{"bootstrap", "--username", "other", "--password-stdin"})
	if err := cmd.Execute(); !errors.Is(err, ErrBootstrapCompleted) {
		t.Fatalf("second bootstrap error = %v", err)
	}
}

type memoryBootstrap struct {
	completed bool
}

func (repository *memoryBootstrap) Bootstrap(_ context.Context, account identity.Account) (identity.Account, error) {
	if repository.completed {
		return identity.Account{}, ErrBootstrapCompleted
	}
	repository.completed = true
	account.ID = "platform-admin"
	account.PlatformAdmin = true
	return account, nil
}
