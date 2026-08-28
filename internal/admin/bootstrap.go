package admin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/TashanGKD/tashan-compute/internal/identity"
)

var ErrBootstrapCompleted = identity.ErrBootstrapCompleted

type BootstrapRepository interface {
	Bootstrap(context.Context, identity.Account) (identity.Account, error)
}

func bootstrap(ctx context.Context, dependencies Dependencies, username string, passwordStdin bool) (identity.Account, error) {
	password, err := readBootstrapPassword(dependencies, passwordStdin)
	if err != nil {
		return identity.Account{}, err
	}
	repository := dependencies.Repository
	cleanup := func() {}
	if repository == nil && dependencies.RepositoryFactory != nil {
		repository, cleanup, err = dependencies.RepositoryFactory(ctx)
		if err != nil {
			return identity.Account{}, err
		}
		defer cleanup()
	}
	if repository == nil {
		return identity.Account{}, errors.New("bootstrap repository is unavailable")
	}
	hash, err := dependencies.Hasher.Hash(password, username)
	if err != nil {
		return identity.Account{}, err
	}
	return repository.Bootstrap(ctx, identity.Account{
		Username:           username,
		PasswordHash:       hash,
		PlatformAdmin:      true,
		PasswordVersion:    1,
		MustChangePassword: false,
	})
}

func readBootstrapPassword(dependencies Dependencies, passwordStdin bool) (string, error) {
	if passwordStdin {
		reader := dependencies.Stdin
		if reader == nil {
			return "", errors.New("stdin is unavailable")
		}
		line, err := bufio.NewReader(io.LimitReader(reader, 2048)).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return "", fmt.Errorf("read password from stdin: %w", err)
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	if dependencies.IsTerminal == nil || !dependencies.IsTerminal() {
		return "", errors.New("interactive terminal or --password-stdin is required")
	}
	if dependencies.ReadPassword == nil {
		return "", errors.New("hidden password reader is unavailable")
	}
	first, err := dependencies.ReadPassword("Initial platform administrator password: ")
	if err != nil {
		return "", err
	}
	second, err := dependencies.ReadPassword("Confirm password: ")
	if err != nil {
		return "", err
	}
	if first != second {
		return "", errors.New("password confirmation does not match")
	}
	return first, nil
}
