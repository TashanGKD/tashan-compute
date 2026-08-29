package codercli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFindBinaryPrefersVerifiedSibling(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "tcompute")
	name := "coder"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	coder := filepath.Join(dir, name)
	if err := os.WriteFile(coder, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := FindBinary(executable, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != coder {
		t.Fatalf("binary = %q, want %q", got, coder)
	}
}

func TestFindBinaryRejectsRelativeOverrideAndDirectory(t *testing.T) {
	if _, err := FindBinary("/opt/tcompute", "./coder"); err == nil {
		t.Fatal("relative override accepted")
	}
	if _, err := FindBinary("/opt/tcompute", t.TempDir()); err == nil {
		t.Fatal("directory override accepted")
	}
}
