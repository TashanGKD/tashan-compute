package codercli

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"
)

type captureExecutor struct {
	path string
	args []string
	env  []string
}

func (executor *captureExecutor) Run(_ context.Context, path string, args, env []string, _ io.Reader, _, _ io.Writer) error {
	executor.path = path
	executor.args = append([]string(nil), args...)
	executor.env = append([]string(nil), env...)
	return nil
}

func TestRunnerUsesArgvAndEnvironmentToken(t *testing.T) {
	executor := &captureExecutor{}
	runner := Runner{Binary: "/opt/tcompute/coder", Executor: executor, BaseURL: "https://compute.tashan.chat"}
	var stdout, stderr bytes.Buffer
	if err := runner.Run(context.Background(), "session-secret", []string{"ssh", "space-a", "--", "printf", "%s", "hello; touch /tmp/pwn"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if executor.path != "/opt/tcompute/coder" {
		t.Fatalf("path = %q", executor.path)
	}
	want := []string{"ssh", "space-a", "--", "printf", "%s", "hello; touch /tmp/pwn"}
	if !reflect.DeepEqual(executor.args, want) {
		t.Fatalf("args = %#v", executor.args)
	}
	for _, arg := range executor.args {
		if strings.Contains(arg, "session-secret") {
			t.Fatal("token leaked into argv")
		}
	}
	joined := strings.Join(executor.env, "\n")
	if !strings.Contains(joined, "CODER_SESSION_TOKEN=session-secret") || !strings.Contains(joined, "CODER_URL=https://compute.tashan.chat") {
		t.Fatalf("missing Coder env: %s", joined)
	}
}

func TestRunnerRejectsUnsafeBinaryAndToken(t *testing.T) {
	for _, tc := range []struct {
		name   string
		runner Runner
		token  string
	}{
		{name: "relative binary", runner: Runner{Binary: "./coder", Executor: &captureExecutor{}, BaseURL: "https://compute.tashan.chat"}, token: "token"},
		{name: "empty token", runner: Runner{Binary: "/opt/coder", Executor: &captureExecutor{}, BaseURL: "https://compute.tashan.chat"}},
		{name: "newline token", runner: Runner{Binary: "/opt/coder", Executor: &captureExecutor{}, BaseURL: "https://compute.tashan.chat"}, token: "token\ninjected=x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.runner.Run(context.Background(), tc.token, []string{"list"}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
				t.Fatal("unsafe runner input accepted")
			}
		})
	}
}
