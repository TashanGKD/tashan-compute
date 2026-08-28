package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/spf13/cobra"
)

const sessionCredentialLabel = "default-session"

type LoginClient interface {
	Login(context.Context, httpapi.LoginInput) (httpapi.LoginResult, error)
}

type storedSession struct {
	AccessToken     string    `json:"access_token"`
	RefreshToken    string    `json:"refresh_token"`
	AccessExpiresAt time.Time `json:"access_expires_at"`
}

func newAuthCommand(dependencies Dependencies) *cobra.Command {
	authCommand := &cobra.Command{Use: "auth", Short: "Authenticate this device"}
	var username, deviceLabel, deviceFingerprint string
	var passwordStdin bool
	loginCommand := &cobra.Command{
		Use:   "login",
		Short: "Log in with an administrator-created account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dependencies.LoginClient == nil || dependencies.CredentialStore == nil {
				return errors.New("login dependencies are unavailable")
			}
			password, err := readPasswordFromInput(dependencies, passwordStdin)
			if err != nil {
				return err
			}
			result, err := dependencies.LoginClient.Login(cmd.Context(), httpapi.LoginInput{
				Username: username, Password: password, DeviceLabel: deviceLabel, DeviceFingerprint: deviceFingerprint,
			})
			if err != nil {
				return err
			}
			encoded, err := json.Marshal(storedSession{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, AccessExpiresAt: result.AccessExpiresAt})
			if err != nil {
				return fmt.Errorf("encode session credentials: %w", err)
			}
			if err := dependencies.CredentialStore.Save(cmd.Context(), sessionCredentialLabel, string(encoded)); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "login succeeded; initial password change required: %t\n", result.MustChangePassword)
			return nil
		},
	}
	loginCommand.Flags().StringVar(&username, "username", "", "administrator-created username")
	loginCommand.Flags().StringVar(&deviceLabel, "device-label", "", "human-readable device label")
	loginCommand.Flags().StringVar(&deviceFingerprint, "device-fingerprint", "", "stable local device fingerprint")
	loginCommand.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read password from stdin")
	_ = loginCommand.MarkFlagRequired("username")
	_ = loginCommand.MarkFlagRequired("device-label")
	_ = loginCommand.MarkFlagRequired("device-fingerprint")
	authCommand.AddCommand(loginCommand)
	addAuthSessionCommands(authCommand, dependencies)
	return authCommand
}

func readPasswordFromInput(dependencies Dependencies, fromStdin bool) (string, error) {
	if !fromStdin {
		if dependencies.IsTerminal == nil || !dependencies.IsTerminal() || dependencies.ReadPassword == nil {
			return "", errors.New("interactive hidden input is unavailable; use --password-stdin")
		}
		return dependencies.ReadPassword("Password: ")
	}
	input := dependencies.Stdin
	if input == nil {
		return "", errors.New("stdin is unavailable")
	}
	line, err := bufio.NewReader(io.LimitReader(input, 2048)).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read password: %w", err)
	}
	password := strings.TrimRight(line, "\r\n")
	if password == "" {
		return "", errors.New("password must not be empty")
	}
	return password, nil
}
