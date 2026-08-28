package credentials

import (
	"context"
	"errors"
	"strings"
)

type MacOSKeychainStore struct {
	runner CommandRunner
}

func NewMacOSKeychainStore(runner CommandRunner) *MacOSKeychainStore {
	return &MacOSKeychainStore{runner: runner}
}

func (store *MacOSKeychainStore) Save(ctx context.Context, label, secret string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	if err := validateSecret(secret); err != nil {
		return err
	}
	result, err := store.runner.Run(ctx, "/usr/bin/security", []string{"add-generic-password", "-U", "-a", label, "-s", serviceName, "-w"}, secret)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return errors.New("macOS Keychain write failed")
	}
	return nil
}

func (store *MacOSKeychainStore) Load(ctx context.Context, label string) (string, bool, error) {
	if err := validateLabel(label); err != nil {
		return "", false, err
	}
	result, err := store.runner.Run(ctx, "/usr/bin/security", []string{"find-generic-password", "-a", label, "-s", serviceName, "-w"}, "")
	if err != nil {
		return "", false, err
	}
	if result.ExitCode == 44 {
		return "", false, nil
	}
	if result.ExitCode != 0 {
		return "", false, errors.New("macOS Keychain read failed")
	}
	return strings.TrimRight(result.Stdout, "\r\n"), true, nil
}

func (store *MacOSKeychainStore) Delete(ctx context.Context, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	result, err := store.runner.Run(ctx, "/usr/bin/security", []string{"delete-generic-password", "-a", label, "-s", serviceName}, "")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 && result.ExitCode != 44 {
		return errors.New("macOS Keychain delete failed")
	}
	return nil
}
