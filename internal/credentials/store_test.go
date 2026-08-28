package credentials

import (
	"context"
	"strings"
	"testing"
)

func TestMemoryStoreRoundTrip(t *testing.T) {
	store := NewMemoryStore()
	if err := store.Save(context.Background(), "session", "fixture-secret"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	value, found, err := store.Load(context.Background(), "session")
	if err != nil || !found || value != "fixture-secret" {
		t.Fatalf("Load() = %q, %v, %v", value, found, err)
	}
	if err := store.Delete(context.Background(), "session"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestMacOSKeychainWritesSecretThroughStdinNotArguments(t *testing.T) {
	runner := &recordingRunner{}
	store := NewMacOSKeychainStore(runner)
	if err := store.Save(context.Background(), "session", "fixture-secret"); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if strings.Contains(strings.Join(runner.args, " "), "fixture-secret") {
		t.Fatalf("secret leaked into argv: %v", runner.args)
	}
	if runner.stdin != "fixture-secret" {
		t.Fatalf("stdin = %q", runner.stdin)
	}
	if len(runner.args) == 0 || runner.args[len(runner.args)-1] != "-w" {
		t.Fatalf("security -w is not final argument: %v", runner.args)
	}
}

func TestCredentialLabelsRejectPathAndControlInput(t *testing.T) {
	store := NewMemoryStore()
	for _, label := range []string{"../session", "session\nother", "", strings.Repeat("x", 129)} {
		if err := store.Save(context.Background(), label, "fixture-secret"); err == nil {
			t.Fatalf("Save(%q) error = nil", label)
		}
	}
}

type recordingRunner struct {
	path  string
	args  []string
	stdin string
}

func (runner *recordingRunner) Run(_ context.Context, path string, args []string, stdin string) (CommandResult, error) {
	runner.path = path
	runner.args = append([]string(nil), args...)
	runner.stdin = stdin
	return CommandResult{}, nil
}
