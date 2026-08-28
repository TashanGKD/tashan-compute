package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxScannedFileSize = 5 << 20

var (
	privateKeyPattern = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)
	bearerPattern     = regexp.MustCompile(`(?i)Bearer[[:space:]]+[A-Za-z0-9._~-]{20,}`)
	databasePattern   = regexp.MustCompile(`(?i)postgres(?:ql)?://[^:/[:space:]]+:[^@[:space:]]+@`)
	envSecretPattern  = regexp.MustCompile(`(?:^|[[:space:]])[A-Z0-9_]*(?:PASSWORD|SECRET|TOKEN)[A-Z0-9_]*[[:space:]]*=[[:space:]]*["']?([^"'[:space:]]+)`)
	jsonSecretPattern = regexp.MustCompile(`(?i)["'](?:password|secret|token|private_key)["'][[:space:]]*:[[:space:]]*["']([^"']+)`)
)

func main() {
	root := flag.String("root", ".", "repository or fixture root")
	flag.Parse()
	if err := scan(*root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func scan(root string) error {
	filesScanned := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".worktrees" || entry.Name() == "dist" || entry.Name() == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > maxScannedFileSize {
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if bytes.IndexByte(contents, 0) >= 0 {
			return nil
		}
		filesScanned++
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if privateKeyPattern.Match(contents) {
			return fmt.Errorf("%s: private key material", relative)
		}
		for _, line := range strings.Split(string(contents), "\n") {
			if safeExample(line) {
				continue
			}
			if bearerPattern.MatchString(line) {
				return fmt.Errorf("%s: bearer token", relative)
			}
			if databasePattern.MatchString(line) {
				return fmt.Errorf("%s: credential-bearing database URL", relative)
			}
			if matches := envSecretPattern.FindStringSubmatch(line); len(matches) == 2 && matches[1] != "" {
				return fmt.Errorf("%s: password assignment", relative)
			}
			if matches := jsonSecretPattern.FindStringSubmatch(line); len(matches) == 2 && matches[1] != "" {
				return fmt.Errorf("%s: password assignment", relative)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("public repository secret scan: PASS (%d text files)\n", filesScanned)
	return nil
}

func safeExample(line string) bool {
	lower := strings.ToLower(line)
	for _, marker := range []string{"fixture", "test_only", "tcompute_test", "[redacted]", "<redacted>", "example.test"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	trimmed := strings.TrimSpace(line)
	return strings.HasSuffix(trimmed, "=") || trimmed == ""
}
