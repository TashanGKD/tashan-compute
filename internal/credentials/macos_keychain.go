package credentials

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const macOSKeychainEncodingPrefix = "go-keyring-base64:"

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
	encoded := macOSKeychainEncodingPrefix + base64.StdEncoding.EncodeToString([]byte(secret))
	interactiveCommand := fmt.Sprintf("add-generic-password -U -a %s -s %s -w %s\n", label, serviceName, encoded)
	result, err := store.runner.Run(ctx, "/usr/bin/security", []string{"-i"}, interactiveCommand)
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
	value := strings.TrimRight(result.Stdout, "\r\n")
	if strings.HasPrefix(value, macOSKeychainEncodingPrefix) {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, macOSKeychainEncodingPrefix))
		if err != nil {
			return "", false, errors.New("macOS Keychain value is invalid")
		}
		value = string(decoded)
	}
	return value, true, nil
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
