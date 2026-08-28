package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootWithoutArgumentsOnlyPrintsHelp(t *testing.T) {
	called := false
	cmd := NewRoot(Dependencies{NetworkProbe: func() { called = true }})
	cmd.SetArgs(nil)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called {
		t.Fatal("no-argument execution accessed the network")
	}
	if !strings.Contains(stdout.String(), "Tashan Compute") {
		t.Fatalf("stdout = %q, want help containing Tashan Compute", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
