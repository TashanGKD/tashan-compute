package credentials

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

const serviceName = "chat.tashan.compute"

var labelPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,128}$`)

type Store interface {
	Save(context.Context, string, string) error
	Load(context.Context, string) (string, bool, error)
	Delete(context.Context, string) error
}

type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type CommandRunner interface {
	Run(context.Context, string, []string, string) (CommandResult, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, path string, args []string, stdin string) (CommandResult, error) {
	command := exec.CommandContext(ctx, path, args...)
	command.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := CommandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
		return result, nil
	}
	return CommandResult{}, fmt.Errorf("run credential helper: %w", err)
}

func validateLabel(label string) error {
	if !labelPattern.MatchString(label) {
		return errors.New("credential label must contain 1-128 safe characters")
	}
	return nil
}

func validateSecret(secret string) error {
	if secret == "" {
		return errors.New("credential secret must not be empty")
	}
	return nil
}
