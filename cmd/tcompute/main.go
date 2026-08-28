package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/TashanGKD/tashan-compute/internal/cli"
	"github.com/TashanGKD/tashan-compute/internal/client"
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
	apiURL := os.Getenv("TCOMPUTE_API_URL")
	if apiURL == "" {
		apiURL = "http://127.0.0.1:8180"
	}
	api, err := client.New(apiURL, http.DefaultClient)
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
		store = credentials.NewMemoryStore()
	}
	return cli.Dependencies{
		LoginClient: api, APIClient: api, CredentialStore: store, Stdin: os.Stdin,
		IsTerminal: func() bool { return term.IsTerminal(int(os.Stdin.Fd())) },
		ReadPassword: func(prompt string) (string, error) {
			fmt.Fprint(os.Stderr, prompt)
			value, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			return string(value), err
		},
	}, nil
}
