package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TashanGKD/tashan-compute/internal/credentials"
)

func TestAdminUserCreateUsesStoredSessionAndProtectedPasswordInput(t *testing.T) {
	store := credentials.NewMemoryStore()
	encoded, _ := json.Marshal(storedSession{AccessToken: "fixture-access", RefreshToken: "fixture-refresh"})
	if err := store.Save(context.Background(), sessionCredentialLabel, string(encoded)); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	api := &recordingAPIClient{}
	cmd := NewRoot(Dependencies{APIClient: api, CredentialStore: store, Stdin: bytes.NewBufferString("bob initial fixture password\n")})
	cmd.SetArgs([]string{"admin", "user", "create", "--username", "bob", "--password-stdin"})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if api.path != "/v1/admin/users" || api.accessToken != "fixture-access" {
		t.Fatalf("API call = %s token=%s", api.path, api.accessToken)
	}
	output := stdout.String() + stderr.String()
	if strings.Contains(output, "bob initial fixture password") || strings.Contains(output, "fixture-access") {
		t.Fatalf("output leaked secret: %s", output)
	}
}

func TestAdminPasswordResetRequiresProtectedInput(t *testing.T) {
	store := credentials.NewMemoryStore()
	encoded, _ := json.Marshal(storedSession{AccessToken: "fixture-access", RefreshToken: "fixture-refresh"})
	_ = store.Save(context.Background(), sessionCredentialLabel, string(encoded))
	api := &recordingAPIClient{}
	cmd := NewRoot(Dependencies{APIClient: api, CredentialStore: store})
	cmd.SetArgs([]string{"admin", "user", "reset-password", "account-1", "--password", "leaked"})
	if err := cmd.Execute(); err == nil || err.Error() != "unknown flag: --password" {
		t.Fatalf("--password error = %v", err)
	}
	cmd = NewRoot(Dependencies{APIClient: api, CredentialStore: store, Stdin: bytes.NewBufferString("new fixture password 2026\n")})
	cmd.SetArgs([]string{"admin", "user", "reset-password", "account-1", "--password-stdin"})
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("reset-password error = %v", err)
	}
	if api.path != "/v1/admin/users/account-1/reset-password" {
		t.Fatalf("path = %s", api.path)
	}
}

type recordingAPIClient struct {
	method      string
	path        string
	accessToken string
	input       any
}

func (client *recordingAPIClient) Do(_ context.Context, method, path, accessToken string, input, output any) error {
	client.method = method
	client.path = path
	client.accessToken = accessToken
	client.input = input
	if destination, ok := output.(*map[string]any); ok {
		*destination = map[string]any{"ok": true}
	}
	return nil
}
