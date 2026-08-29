package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/TashanGKD/tashan-compute/internal/cli"
	"github.com/TashanGKD/tashan-compute/internal/codercli"
	"github.com/TashanGKD/tashan-compute/internal/credentials"
	"golang.org/x/term"
)

func main() {
	dependencies, err := runtimeDependencies()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := cli.NewRoot(dependencies).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runtimeDependencies() (cli.Dependencies, error) {
	if len(os.Args) == 1 {
		return cli.Dependencies{}, nil
	}
	coderURL := os.Getenv("TCOMPUTE_CODER_URL")
	if coderURL == "" {
		coderURL = "https://compute.tashan.chat"
	}
	coderClient, err := codercli.NewClient(coderURL, http.DefaultClient)
	if err != nil {
		return cli.Dependencies{}, err
	}
	executable, err := os.Executable()
	if err != nil {
		return cli.Dependencies{}, fmt.Errorf("resolve tcompute executable: %w", err)
	}
	coderBinary, err := codercli.FindBinary(executable, os.Getenv("TCOMPUTE_CODER_BIN"))
	if err != nil {
		return cli.Dependencies{}, err
	}
	runner := credentials.ExecRunner{}
	var store credentials.Store
	switch runtime.GOOS {
	case "darwin":
		store = credentials.NewMacOSKeychainStore(runner)
	case "linux":
		store = credentials.NewLinuxSecretServiceStore(runner)
	default:
		return cli.Dependencies{}, fmt.Errorf("secure credential storage is not implemented for %s", runtime.GOOS)
	}
	coderRunner := codercli.Runner{Binary: coderBinary, Executor: codercli.ProcessExecutor{}, BaseURL: coderURL}
	return cli.Dependencies{
		CoderClient:     coderClient,
		CoderRunner:     coderRunner,
		CoderSyncer:     coderRunner,
		CredentialStore: store,
		Stdin:           os.Stdin,
		IsTerminal:      func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		ReadPassword: func(prompt string) (string, error) {
			fmt.Fprint(os.Stderr, prompt)
			value, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			return string(value), err
		},
	}, nil
}
