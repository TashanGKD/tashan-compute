package codercli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func FindBinary(tcomputeExecutable, override string) (string, error) {
	if override != "" {
		if !filepath.IsAbs(override) {
			return "", errors.New("TCOMPUTE_CODER_BIN must be an absolute path")
		}
		if err := validateExecutable(override); err != nil {
			return "", err
		}
		return filepath.Clean(override), nil
	}

	name := "coder"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if filepath.IsAbs(tcomputeExecutable) {
		resolvedExecutable := tcomputeExecutable
		if resolved, err := filepath.EvalSymlinks(tcomputeExecutable); err == nil {
			resolvedExecutable = resolved
		}
		sibling := filepath.Join(filepath.Dir(resolvedExecutable), name)
		if validateExecutable(sibling) == nil {
			return sibling, nil
		}
	}

	found, err := exec.LookPath(name)
	if err != nil {
		return "", errors.New("Coder CLI not found beside tcompute or on PATH")
	}
	absolute, err := filepath.Abs(found)
	if err != nil {
		return "", errors.New("resolve Coder CLI path")
	}
	if err := validateExecutable(absolute); err != nil {
		return "", err
	}
	return absolute, nil
}

func validateExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return errors.New("Coder CLI path is unavailable")
	}
	if !info.Mode().IsRegular() {
		return errors.New("Coder CLI path must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return errors.New("Coder CLI path is not executable")
	}
	return nil
}
