package codercli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type Executor interface {
	Run(context.Context, string, []string, []string, io.Reader, io.Writer, io.Writer) error
}

type ProcessExecutor struct{}

func (ProcessExecutor) Run(ctx context.Context, path string, args, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	command := exec.CommandContext(ctx, path, args...)
	command.Env = env
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return fmt.Errorf("Coder CLI exited with status %d", exitError.ExitCode())
		}
		return errors.New("failed to execute Coder CLI")
	}
	return nil
}

type Runner struct {
	Binary   string
	Executor Executor
	BaseURL  string
}

type SyncRequest struct {
	Direction  string
	Workspace  string
	LocalPath  string
	RemotePath string
	Apply      bool
}

var syncWorkspacePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}(?:/[a-z0-9][a-z0-9-]{0,62})?$`)

func (runner Runner) Run(ctx context.Context, token string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if !filepath.IsAbs(runner.Binary) || runner.Executor == nil {
		return errors.New("Coder CLI path must be absolute")
	}
	if token == "" || strings.ContainsAny(token, "\x00\r\n") {
		return errors.New("stored Coder session is invalid")
	}
	if _, err := validateBaseURL(runner.BaseURL); err != nil {
		return err
	}
	for _, arg := range args {
		if strings.ContainsRune(arg, '\x00') {
			return errors.New("Coder CLI argument contains a null byte")
		}
	}
	env := safeEnvironment(os.Environ())
	env = append(env, "CODER_URL="+runner.BaseURL, "CODER_SESSION_TOKEN="+token, "CODER_USE_KEYRING=true")
	return runner.Executor.Run(ctx, runner.Binary, args, env, stdin, stdout, stderr)
}

func (runner Runner) Sync(ctx context.Context, token string, request SyncRequest, stdout, stderr io.Writer) error {
	if !filepath.IsAbs(runner.Binary) || runner.Executor == nil || strings.ContainsAny(runner.Binary, "\"\r\n") {
		return errors.New("Coder CLI path is unsafe")
	}
	if token == "" || strings.ContainsAny(token, "\x00\r\n") {
		return errors.New("stored Coder session is invalid")
	}
	if !syncWorkspacePattern.MatchString(request.Workspace) {
		return errors.New("invalid workspace reference")
	}
	if request.LocalPath == "" || strings.ContainsAny(request.LocalPath, "\x00\r\n") {
		return errors.New("local path is required")
	}
	remotePath := path.Clean(request.RemotePath)
	if request.RemotePath == "" || remotePath == "." || path.IsAbs(remotePath) || remotePath == ".." || strings.HasPrefix(remotePath, "../") {
		return errors.New("remote path must stay below /home/coder")
	}

	configFile, err := os.CreateTemp("", "tcompute-ssh-config-*")
	if err != nil {
		return errors.New("create temporary SSH config")
	}
	configPath := configFile.Name()
	defer os.Remove(configPath)
	config := fmt.Sprintf("Host tcompute-sync\n  ProxyCommand \"%s\" ssh --stdio \"%s\"\n  StrictHostKeyChecking no\n  UserKnownHostsFile /dev/null\n", runner.Binary, request.Workspace)
	if _, err := io.WriteString(configFile, config); err != nil {
		configFile.Close()
		return errors.New("write temporary SSH config")
	}
	if err := configFile.Chmod(0o600); err != nil {
		configFile.Close()
		return errors.New("secure temporary SSH config")
	}
	if err := configFile.Close(); err != nil {
		return errors.New("close temporary SSH config")
	}

	rsyncArgs := []string{"-a", "--partial", "--stats"}
	if !request.Apply {
		rsyncArgs = append(rsyncArgs, "--dry-run")
	}
	rsyncArgs = append(rsyncArgs, "-e", "ssh -F "+configPath)
	remote := "tcompute-sync:/home/coder/" + remotePath
	switch request.Direction {
	case "push":
		rsyncArgs = append(rsyncArgs, request.LocalPath, remote)
	case "pull":
		rsyncArgs = append(rsyncArgs, remote, request.LocalPath)
	default:
		return errors.New("sync direction must be push or pull")
	}
	env := safeEnvironment(os.Environ())
	env = append(env, "CODER_URL="+runner.BaseURL, "CODER_SESSION_TOKEN="+token, "CODER_USE_KEYRING=true")
	return runner.Executor.Run(ctx, "/usr/bin/rsync", rsyncArgs, env, strings.NewReader(""), stdout, stderr)
}

func safeEnvironment(source []string) []string {
	allowed := map[string]bool{
		"HOME": true, "PATH": true, "TMPDIR": true, "TERM": true, "LANG": true,
		"SSH_AUTH_SOCK": true, "DBUS_SESSION_BUS_ADDRESS": true,
		"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true,
		"http_proxy": true, "https_proxy": true, "no_proxy": true,
	}
	result := make([]string, 0, len(source))
	for _, entry := range source {
		key, _, _ := strings.Cut(entry, "=")
		if allowed[key] || strings.HasPrefix(key, "LC_") || strings.HasPrefix(key, "XDG_") {
			result = append(result, entry)
		}
	}
	return result
}
