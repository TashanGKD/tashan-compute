package codercli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

type multiCaptureExecutor struct{ calls [][]string }

func (executor *multiCaptureExecutor) Run(_ context.Context, path string, args, _ []string, _ io.Reader, _, _ io.Writer) error {
	executor.calls = append(executor.calls, append([]string{path}, args...))
	return nil
}

func TestSyncDefaultsToDryRunAndNeverDeletes(t *testing.T) {
	executor := &multiCaptureExecutor{}
	runner := Runner{Binary: "/opt/tcompute/coder", Executor: executor, BaseURL: "https://compute.tashan.chat"}
	if err := runner.Sync(context.Background(), "session-token", SyncRequest{Direction: "push", Workspace: "space-a", LocalPath: "/tmp/source", RemotePath: "project"}, &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if len(executor.calls) != 1 {
		t.Fatalf("calls = %#v", executor.calls)
	}
	rsync := strings.Join(executor.calls[0], " ")
	for _, required := range []string{"--dry-run", "/tmp/source", "tcompute-sync:/home/coder/project"} {
		if !strings.Contains(rsync, required) {
			t.Fatalf("rsync missing %q: %s", required, rsync)
		}
	}
	if strings.Contains(rsync, "--delete") {
		t.Fatalf("rsync contains delete: %s", rsync)
	}
	if !strings.Contains(rsync, " -- /tmp/source tcompute-sync:/home/coder/project") {
		t.Fatalf("rsync has no option terminator: %s", rsync)
	}
}

func TestSyncRejectsLeadingDashLocalPath(t *testing.T) {
	runner := Runner{Binary: "/opt/tcompute/coder", Executor: &multiCaptureExecutor{}, BaseURL: "https://compute.tashan.chat"}
	if err := runner.Sync(context.Background(), "token", SyncRequest{Direction: "push", Workspace: "space-a", LocalPath: "--delete", RemotePath: "project"}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Fatal("leading-dash local path accepted")
	}
}

func TestSyncRejectsRemotePathEscapes(t *testing.T) {
	runner := Runner{Binary: "/opt/tcompute/coder", Executor: &multiCaptureExecutor{}, BaseURL: "https://compute.tashan.chat"}
	for _, remote := range []string{"", "/etc", "../etc", "a/../../etc", "."} {
		if err := runner.Sync(context.Background(), "token", SyncRequest{Direction: "push", Workspace: "space-a", LocalPath: "/tmp/source", RemotePath: remote}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted remote path %q", remote)
		}
	}
}
