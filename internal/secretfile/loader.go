package secretfile

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func ReadBase64(path string, expectedLength int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect secret file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("secret file must be a regular file, not a symlink")
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("secret file must not be accessible by group or other users")
	}
	if info.Size() <= 0 || info.Size() > 8192 {
		return nil, errors.New("secret file size is invalid")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open secret file: %w", err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil || !os.SameFile(info, openedInfo) {
		return nil, errors.New("secret file changed while opening")
	}
	contents, err := io.ReadAll(io.LimitReader(file, 8193))
	if err != nil || len(contents) > 8192 {
		return nil, errors.New("read secret file failed or exceeded limit")
	}
	encoded := strings.TrimSpace(string(contents))
	decoded, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.StdEncoding.DecodeString(encoded)
	}
	if err != nil || len(decoded) != expectedLength {
		return nil, fmt.Errorf("secret must decode to exactly %d bytes", expectedLength)
	}
	return decoded, nil
}
