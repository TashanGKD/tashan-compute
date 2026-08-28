package credentials

import (
	"context"
	"errors"
	"strings"
)

type LinuxSecretServiceStore struct {
	runner CommandRunner
}

func NewLinuxSecretServiceStore(runner CommandRunner) *LinuxSecretServiceStore {
	return &LinuxSecretServiceStore{runner: runner}
}

func (store *LinuxSecretServiceStore) Save(ctx context.Context, label, secret string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	if err := validateSecret(secret); err != nil {
		return err
	}
	result, err := store.runner.Run(ctx, "secret-tool", []string{"store", "--label=Tashan Compute", "service", serviceName, "account", label}, secret)
	if err != nil {
		return err
	}
	if result.ExitCode != 0 {
		return errors.New("Secret Service write failed")
	}
	return nil
}

func (store *LinuxSecretServiceStore) Load(ctx context.Context, label string) (string, bool, error) {
	if err := validateLabel(label); err != nil {
		return "", false, err
	}
	result, err := store.runner.Run(ctx, "secret-tool", []string{"lookup", "service", serviceName, "account", label}, "")
	if err != nil {
		return "", false, err
	}
	if result.ExitCode == 1 {
		return "", false, nil
	}
	if result.ExitCode != 0 {
		return "", false, errors.New("Secret Service read failed")
	}
	return strings.TrimRight(result.Stdout, "\r\n"), true, nil
}

func (store *LinuxSecretServiceStore) Delete(ctx context.Context, label string) error {
	if err := validateLabel(label); err != nil {
		return err
	}
	result, err := store.runner.Run(ctx, "secret-tool", []string{"clear", "service", serviceName, "account", label}, "")
	if err != nil {
		return err
	}
	if result.ExitCode != 0 && result.ExitCode != 1 {
		return errors.New("Secret Service delete failed")
	}
	return nil
}
