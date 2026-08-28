package secretfile

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestReadBase64RejectsWeakPermissionsAndSymlink(t *testing.T) {
	directory := t.TempDir()
	secret := filepath.Join(directory, "secret")
	encoded := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	if err := os.WriteFile(secret, []byte(encoded+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := ReadBase64(secret, 32); err == nil {
		t.Fatal("ReadBase64() accepted world-readable file")
	}
	if err := os.Chmod(secret, 0o600); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}
	if _, err := ReadBase64(link, 32); err == nil {
		t.Fatal("ReadBase64() accepted symlink")
	}
	value, err := ReadBase64(secret, 32)
	if err != nil || len(value) != 32 {
		t.Fatalf("ReadBase64() = %d bytes, %v", len(value), err)
	}
}
