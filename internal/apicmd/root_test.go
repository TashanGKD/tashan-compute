package apicmd

import (
	"bytes"
	"context"
	"testing"
)

func TestNoArgumentsPrintHelpWithoutBuildingRuntime(t *testing.T) {
	called := false
	cmd := NewRoot(Dependencies{Build: func(context.Context) (Runtime, error) {
		called = true
		return fixtureRuntime{}, nil
	}})
	cmd.SetArgs(nil)
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if called {
		t.Fatal("no-argument help built the API runtime")
	}
}

func TestServeBuildsAndRunsRuntime(t *testing.T) {
	served := false
	cmd := NewRoot(Dependencies{Build: func(context.Context) (Runtime, error) {
		return fixtureRuntime{served: &served}, nil
	}})
	cmd.SetArgs([]string{"serve"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !served {
		t.Fatal("serve did not run API runtime")
	}
}

type fixtureRuntime struct{ served *bool }

func (runtime fixtureRuntime) Serve(context.Context) error {
	if runtime.served != nil {
		*runtime.served = true
	}
	return nil
}
