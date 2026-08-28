package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/credentials"
	"github.com/TashanGKD/tashan-compute/internal/httpapi"
)

func TestLoginRejectsPasswordArgumentAndNeverPrintsTokens(t *testing.T) {
	api := &fixtureLoginClient{}
	store := credentials.NewMemoryStore()
	cmd := NewRoot(Dependencies{LoginClient: api, CredentialStore: store})
	cmd.SetArgs([]string{"auth", "login", "--username", "alice", "--password", "leaked"})
	if err := cmd.Execute(); err == nil || err.Error() != "unknown flag: --password" {
		t.Fatalf("--password error = %v", err)
	}

	cmd = NewRoot(Dependencies{LoginClient: api, CredentialStore: store, Stdin: bytes.NewBufferString("fixture-password\n")})
	cmd.SetArgs([]string{"auth", "login", "--username", "alice", "--password-stdin", "--device-label", "mac", "--device-fingerprint", "fp"})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("login error = %v", err)
	}
	output := stdout.String() + stderr.String()
	for _, secret := range []string{"fixture-password", "fixture-access", "fixture-refresh"} {
		if strings.Contains(output, secret) {
			t.Fatalf("output leaked %q: %s", secret, output)
		}
	}
	if _, found, err := store.Load(context.Background(), sessionCredentialLabel); err != nil || !found {
		t.Fatalf("stored session = found %v, error %v", found, err)
	}
}

func TestLoginUsesHiddenPromptWhenTerminalIsAvailable(t *testing.T) {
	api := &fixtureLoginClient{}
	store := credentials.NewMemoryStore()
	cmd := NewRoot(Dependencies{
		LoginClient: api, CredentialStore: store,
		IsTerminal:   func() bool { return true },
		ReadPassword: func(string) (string, error) { return "fixture-password", nil },
	})
	cmd.SetArgs([]string{"auth", "login", "--username", "alice", "--device-label", "mac", "--device-fingerprint", "fp"})
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("login error = %v", err)
	}
}

type fixtureLoginClient struct{}

func (*fixtureLoginClient) Login(_ context.Context, input httpapi.LoginInput) (httpapi.LoginResult, error) {
	return httpapi.LoginResult{
		AccessToken: "fixture-access", RefreshToken: "fixture-refresh",
		AccessExpiresAt: time.Now().Add(time.Minute), MustChangePassword: true,
	}, nil
}
