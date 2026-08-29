package cli

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/TashanGKD/tashan-compute/internal/codercli"
	"github.com/TashanGKD/tashan-compute/internal/credentials"
)

type fixtureCoderClient struct {
	loginEmail    string
	loginPassword string
	logoutToken   string
}

func (client *fixtureCoderClient) Login(_ context.Context, email, password string) (string, error) {
	client.loginEmail, client.loginPassword = email, password
	return "coder-session-secret", nil
}
func (*fixtureCoderClient) Me(context.Context, string) (codercli.User, error) {
	user := codercli.User{Username: "alice", Email: "alice@example.test", Status: "active"}
	user.Roles = append(user.Roles, struct {
		Name string `json:"name"`
	}{Name: "owner"})
	return user, nil
}
func (client *fixtureCoderClient) Logout(_ context.Context, token string) error {
	client.logoutToken = token
	return nil
}
func (*fixtureCoderClient) ChangePassword(context.Context, string, string, string) error { return nil }
func (*fixtureCoderClient) CreateUser(context.Context, string, string, string, string, string) (codercli.User, error) {
	return codercli.User{}, nil
}
func (*fixtureCoderClient) ResetPassword(context.Context, string, string, string) error { return nil }

type fixtureCoderRunner struct {
	args  []string
	calls [][]string
}

func (runner *fixtureCoderRunner) Run(_ context.Context, token string, args []string, _ io.Reader, stdout, _ io.Writer) error {
	if token != "coder-session-secret" {
		panic("wrong token")
	}
	runner.args = append([]string(nil), args...)
	runner.calls = append(runner.calls, append([]string(nil), args...))
	_, _ = io.WriteString(stdout, "runner-output\n")
	return nil
}

func TestCoderLoginUsesProtectedInputAndStoresOnlyToken(t *testing.T) {
	store := credentials.NewMemoryStore()
	client := &fixtureCoderClient{}
	cmd := NewRoot(Dependencies{CoderClient: client, CredentialStore: store, Stdin: bytes.NewBufferString("login-password\n")})
	cmd.SetArgs([]string{"login", "--email", "alice@example.test", "--password-stdin"})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if client.loginEmail != "alice@example.test" || client.loginPassword != "login-password" {
		t.Fatalf("login input = %q %q", client.loginEmail, client.loginPassword)
	}
	stored, found, err := store.Load(context.Background(), coderSessionCredentialLabel)
	if err != nil || !found || stored != "coder-session-secret" {
		t.Fatalf("stored session = %q found=%v err=%v", stored, found, err)
	}
	output := stdout.String() + stderr.String()
	if strings.Contains(output, "login-password") || strings.Contains(output, "coder-session-secret") {
		t.Fatalf("secret leaked: %s", output)
	}
}

func TestCoderCommandsDelegateExactArgv(t *testing.T) {
	tests := []struct {
		args []string
		want []string
	}{
		{args: []string{"shell", "personal-a", "--", "printf", "%s", "hello;touch /tmp/pwn"}, want: []string{"ssh", "personal-a", "--", "printf", "%s", "hello;touch /tmp/pwn"}},
		{args: []string{"personal", "create", "space-a"}, want: []string{"create", "space-a", "--template", "tcompute-standard", "--use-parameter-defaults", "--yes"}},
		{args: []string{"workspace", "list"}, want: []string{"list", "--output", "json"}},
		{args: []string{"org", "list"}, want: []string{"list", "--search", "shared:true", "--output", "json"}},
		{args: []string{"org", "create", "org-a", "--admin", "alice"}, want: []string{"sharing", "add", "org-a", "--user", "alice:admin"}},
		{args: []string{"shell", "alice/shared-a", "--", "true"}, want: []string{"ssh", "alice/shared-a", "--", "true"}},
		{args: []string{"workspace", "stop", "space-a"}, want: []string{"stop", "space-a", "--yes"}},
		{args: []string{"org", "member", "add", "shared-a", "bob"}, want: []string{"sharing", "add", "shared-a", "--user", "bob"}},
		{args: []string{"org", "member", "remove", "shared-a", "bob"}, want: []string{"restart", "shared-a", "--yes"}},
		{args: []string{"service", "public", "shared-a"}, want: []string{"update", "shared-a", "--always-prompt", "--use-parameter-defaults", "--parameter", "service_visibility=public"}},
		{args: []string{"service", "private", "shared-a"}, want: []string{"update", "shared-a", "--always-prompt", "--use-parameter-defaults", "--parameter", "service_visibility=owner"}},
	}
	for _, tc := range tests {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			store := credentials.NewMemoryStore()
			_ = store.Save(context.Background(), coderSessionCredentialLabel, "coder-session-secret")
			runner := &fixtureCoderRunner{}
			cmd := NewRoot(Dependencies{CoderClient: &fixtureCoderClient{}, CoderRunner: runner, CredentialStore: store, Stdin: strings.NewReader("")})
			cmd.SetArgs(tc.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(runner.args, tc.want) {
				t.Fatalf("argv = %#v, want %#v", runner.args, tc.want)
			}
			if len(tc.args) >= 4 && tc.args[0] == "org" && tc.args[2] == "remove" {
				wantFirst := []string{"sharing", "remove", "shared-a", "--user", "bob"}
				if len(runner.calls) != 2 || !reflect.DeepEqual(runner.calls[0], wantFirst) {
					t.Fatalf("remove calls = %#v", runner.calls)
				}
			}
		})
	}
}

func TestWorkspaceDeleteRequiresExplicitYes(t *testing.T) {
	store := credentials.NewMemoryStore()
	_ = store.Save(context.Background(), coderSessionCredentialLabel, "coder-session-secret")
	runner := &fixtureCoderRunner{}
	cmd := NewRoot(Dependencies{CoderRunner: runner, CredentialStore: store})
	cmd.SetArgs([]string{"workspace", "delete", "space-a"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("delete error = %v", err)
	}
}
