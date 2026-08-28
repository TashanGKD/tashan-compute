package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/TashanGKD/tashan-compute/internal/httpapi"
	"github.com/spf13/cobra"
)

type APIClient interface {
	Do(context.Context, string, string, string, any, any) error
}

func newHealthCommand(dependencies Dependencies) *cobra.Command {
	return genericCommand(dependencies, "health", "Read platform health", http.MethodGet, httpapi.RouteHealth, false, nil)
}

func newCapabilityCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "capability", Short: "Inspect server capabilities"}
	root.AddCommand(genericCommand(dependencies, "list", "List server capabilities", http.MethodGet, httpapi.RouteCapabilities, false, nil))
	return root
}

func addAuthSessionCommands(root *cobra.Command, dependencies Dependencies) {
	password := &cobra.Command{Use: "password", Short: "Change account password"}
	password.AddCommand(passwordChangeCommand(dependencies, "initial-change", httpapi.RouteInitialPasswordChange))
	password.AddCommand(passwordChangeCommand(dependencies, "change", httpapi.RoutePasswordChange))
	root.AddCommand(password)

	root.AddCommand(&cobra.Command{Use: "refresh", Short: "Rotate the current device session", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		session, err := loadStoredSession(cmd.Context(), dependencies)
		if err != nil {
			return err
		}
		var result httpapi.LoginResult
		if err := requireAPI(dependencies).Do(cmd.Context(), http.MethodPost, httpapi.RouteRefresh, "", map[string]string{"refresh_token": session.RefreshToken}, &result); err != nil {
			return err
		}
		return saveLoginResult(cmd, dependencies, result)
	}})
	root.AddCommand(&cobra.Command{Use: "logout", Short: "Revoke the current device session", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		session, err := loadStoredSession(cmd.Context(), dependencies)
		if err != nil {
			return err
		}
		var output map[string]any
		if err := requireAPI(dependencies).Do(cmd.Context(), http.MethodPost, httpapi.RouteLogout, session.AccessToken, nil, &output); err != nil {
			return err
		}
		if err := dependencies.CredentialStore.Delete(cmd.Context(), sessionCredentialLabel); err != nil {
			return err
		}
		return writeCommandJSON(cmd, output)
	}})
	root.AddCommand(genericCommand(dependencies, "whoami", "Show the current server identity", http.MethodGet, httpapi.RouteWhoAmI, true, nil))
}

func passwordChangeCommand(dependencies Dependencies, name, path string) *cobra.Command {
	var passwordsStdin bool
	command := &cobra.Command{Use: name, Short: "Change password and revoke all existing sessions", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		current, next, err := readPasswordPair(dependencies, passwordsStdin)
		if err != nil {
			return err
		}
		session, err := loadStoredSession(cmd.Context(), dependencies)
		if err != nil {
			return err
		}
		var output httpapi.PasswordChangeResult
		if err := requireAPI(dependencies).Do(cmd.Context(), http.MethodPost, path, session.AccessToken, httpapi.PasswordChangeInput{CurrentPassword: current, NewPassword: next}, &output); err != nil {
			return err
		}
		if err := dependencies.CredentialStore.Delete(cmd.Context(), sessionCredentialLabel); err != nil {
			return err
		}
		return writeCommandJSON(cmd, output)
	}}
	command.Flags().BoolVar(&passwordsStdin, "passwords-stdin", false, "read current and new password as two stdin lines")
	return command
}

func newDeviceCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "device", Short: "Manage this account's devices"}
	root.AddCommand(genericCommand(dependencies, "list", "List devices", http.MethodGet, httpapi.RouteDevices, true, nil))
	revoke := genericCommand(dependencies, "revoke <device-id>", "Revoke one device", http.MethodDelete, "", true, func(args []string) (string, any, error) {
		return httpapi.RouteDevices + "/" + url.PathEscape(args[0]), nil, nil
	})
	revoke.Args = cobra.ExactArgs(1)
	root.AddCommand(revoke)
	return root
}

func newAdminCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "admin", Short: "Platform administrator operations"}
	users := &cobra.Command{Use: "user", Short: "Manage platform users"}
	var username string
	var passwordStdin bool
	create := &cobra.Command{Use: "create", Short: "Create an ordinary managed account", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		password, err := readPasswordFromInput(dependencies, passwordStdin)
		if err != nil {
			return err
		}
		return authenticatedCall(cmd, dependencies, http.MethodPost, httpapi.RouteAdminUsers, map[string]string{"username": username, "initial_password": password})
	}}
	create.Flags().StringVar(&username, "username", "", "new username")
	create.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read initial password from stdin")
	_ = create.MarkFlagRequired("username")
	users.AddCommand(create)

	var resetPasswordStdin bool
	resetPassword := &cobra.Command{Use: "reset-password <account-id>", Short: "Reset a managed account and revoke its sessions", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		password, err := readPasswordFromInput(dependencies, resetPasswordStdin)
		if err != nil {
			return err
		}
		return authenticatedCall(cmd, dependencies, http.MethodPost, httpapi.RouteAdminUsers+"/"+url.PathEscape(args[0])+"/reset-password", map[string]string{"initial_password": password})
	}}
	resetPassword.Flags().BoolVar(&resetPasswordStdin, "password-stdin", false, "read new initial password from stdin")
	users.AddCommand(resetPassword)
	disable := genericCommand(dependencies, "disable <account-id>", "Disable a managed account", http.MethodPost, "", true, func(args []string) (string, any, error) {
		return httpapi.RouteAdminUsers + "/" + url.PathEscape(args[0]) + "/disable", nil, nil
	})
	disable.Args = cobra.ExactArgs(1)
	users.AddCommand(disable)
	root.AddCommand(users)

	organizations := &cobra.Command{Use: "org", Short: "Manage organizations"}
	var organizationName string
	createOrganization := genericCommand(dependencies, "create", "Create an organization", http.MethodPost, httpapi.RouteAdminOrganizations, true, func(_ []string) (string, any, error) {
		return httpapi.RouteAdminOrganizations, map[string]string{"name": organizationName}, nil
	})
	createOrganization.Flags().StringVar(&organizationName, "name", "", "organization name")
	_ = createOrganization.MarkFlagRequired("name")
	organizations.AddCommand(createOrganization)
	root.AddCommand(organizations)
	return root
}

func newOrganizationCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "org", Short: "Use organizations"}
	root.AddCommand(genericCommand(dependencies, "list", "List organizations", http.MethodGet, httpapi.RouteOrganizations, true, nil))
	members := &cobra.Command{Use: "member", Short: "Manage organization members"}
	for _, definition := range []struct{ name, method string }{{"add", http.MethodPost}, {"remove", http.MethodDelete}, {"role-set", http.MethodPatch}} {
		definition := definition
		var orgID, accountID, role string
		command := genericCommand(dependencies, definition.name, definition.name+" organization member", definition.method, "", true, func(_ []string) (string, any, error) {
			if orgID == "" || accountID == "" {
				return "", nil, errors.New("--org and --account are required")
			}
			body := map[string]string{"account_id": accountID}
			if definition.name == "role-set" {
				if role != "org_admin" && role != "developer" && role != "viewer" {
					return "", nil, errors.New("role must be org_admin, developer, or viewer")
				}
				body["role"] = role
			}
			return httpapi.RouteOrganizations + "/" + url.PathEscape(orgID) + "/members", body, nil
		})
		command.Flags().StringVar(&orgID, "org", "", "organization ID")
		command.Flags().StringVar(&accountID, "account", "", "account ID")
		if definition.name == "role-set" {
			command.Flags().StringVar(&role, "role", "", "org_admin, developer, or viewer")
		}
		members.AddCommand(command)
	}
	root.AddCommand(members)
	return root
}

func newAuditCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "audit", Short: "Inspect authorized audit events"}
	root.AddCommand(genericCommand(dependencies, "list", "List audit events", http.MethodGet, httpapi.RouteAudit, true, nil))
	return root
}

type requestBuilder func([]string) (string, any, error)

func genericCommand(dependencies Dependencies, use, short, method, fixedPath string, authenticated bool, builder requestBuilder) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		path, input := fixedPath, any(nil)
		if builder != nil {
			var err error
			path, input, err = builder(args)
			if err != nil {
				return err
			}
		}
		if authenticated {
			return authenticatedCall(cmd, dependencies, method, path, input)
		}
		var output map[string]any
		if err := requireAPI(dependencies).Do(cmd.Context(), method, path, "", input, &output); err != nil {
			return err
		}
		return writeCommandJSON(cmd, output)
	}}
}

func authenticatedCall(cmd *cobra.Command, dependencies Dependencies, method, path string, input any) error {
	session, err := loadStoredSession(cmd.Context(), dependencies)
	if err != nil {
		return err
	}
	var output map[string]any
	if err := requireAPI(dependencies).Do(cmd.Context(), method, path, session.AccessToken, input, &output); err != nil {
		return err
	}
	return writeCommandJSON(cmd, output)
}

func requireAPI(dependencies Dependencies) APIClient {
	if dependencies.APIClient == nil {
		return unavailableAPIClient{}
	}
	return dependencies.APIClient
}

type unavailableAPIClient struct{}

func (unavailableAPIClient) Do(context.Context, string, string, string, any, any) error {
	return errors.New("API client is unavailable")
}

func loadStoredSession(ctx context.Context, dependencies Dependencies) (storedSession, error) {
	if dependencies.CredentialStore == nil {
		return storedSession{}, errors.New("credential store is unavailable")
	}
	encoded, found, err := dependencies.CredentialStore.Load(ctx, sessionCredentialLabel)
	if err != nil {
		return storedSession{}, err
	}
	if !found {
		return storedSession{}, errors.New("not logged in")
	}
	var session storedSession
	decoder := json.NewDecoder(strings.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&session); err != nil || session.AccessToken == "" || session.RefreshToken == "" {
		return storedSession{}, errors.New("stored session is invalid")
	}
	return session, nil
}

func saveLoginResult(cmd *cobra.Command, dependencies Dependencies, result httpapi.LoginResult) error {
	encoded, err := json.Marshal(storedSession{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, AccessExpiresAt: result.AccessExpiresAt})
	if err != nil {
		return err
	}
	if err := dependencies.CredentialStore.Save(cmd.Context(), sessionCredentialLabel, string(encoded)); err != nil {
		return err
	}
	return writeCommandJSON(cmd, map[string]any{"refreshed": true, "access_expires_at": result.AccessExpiresAt})
}

func writeCommandJSON(cmd *cobra.Command, value any) error {
	return json.NewEncoder(cmd.OutOrStdout()).Encode(value)
}

func readPasswordPair(dependencies Dependencies, fromStdin bool) (string, string, error) {
	if !fromStdin {
		if dependencies.IsTerminal == nil || !dependencies.IsTerminal() || dependencies.ReadPassword == nil {
			return "", "", errors.New("interactive hidden input is unavailable; use --passwords-stdin")
		}
		current, err := dependencies.ReadPassword("Current password: ")
		if err != nil {
			return "", "", err
		}
		next, err := dependencies.ReadPassword("New password: ")
		return current, next, err
	}
	if dependencies.Stdin == nil {
		return "", "", errors.New("stdin is unavailable")
	}
	reader := bufio.NewReader(io.LimitReader(dependencies.Stdin, 4096))
	current, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	next, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", "", err
	}
	current = strings.TrimRight(current, "\r\n")
	next = strings.TrimRight(next, "\r\n")
	if current == "" || next == "" {
		return "", "", errors.New("passwords must not be empty")
	}
	return current, next, nil
}
