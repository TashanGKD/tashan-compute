package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/TashanGKD/tashan-compute/capabilities"
	"github.com/TashanGKD/tashan-compute/internal/codercli"
	"github.com/spf13/cobra"
)

const coderSessionCredentialLabel = "coder-session"

var coderNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}$`)
var coderWorkspaceRefPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}(?:/[a-z0-9][a-z0-9-]{0,62})?$`)

func addCoderCommands(root *cobra.Command, dependencies Dependencies) {
	root.AddCommand(newCoderLoginCommand(dependencies))
	root.AddCommand(newCoderLogoutCommand(dependencies))
	root.AddCommand(newCoderWhoAmICommand(dependencies))
	root.AddCommand(newCoderShellCommand(dependencies))
	root.AddCommand(newCoderPersonalCommand(dependencies))
	root.AddCommand(newCoderWorkspaceCommand(dependencies))
	root.AddCommand(newCoderOrganizationCommand(dependencies))
	root.AddCommand(newCoderServiceCommand(dependencies))
	root.AddCommand(newCoderSyncCommand(dependencies))
	root.AddCommand(newCoderAdminCommand(dependencies))
	root.AddCommand(newCoderCapabilityCommand())
	root.AddCommand(newCoderPasswordCommand(dependencies))
}

func newCoderLoginCommand(dependencies Dependencies) *cobra.Command {
	var email string
	var passwordStdin bool
	command := &cobra.Command{
		Use:   "login",
		Short: "Log in to Tashan Compute with an administrator-created account",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dependencies.CoderClient == nil || dependencies.CredentialStore == nil {
				return errors.New("Coder login dependencies are unavailable")
			}
			password, err := readPasswordFromInput(dependencies, passwordStdin)
			if err != nil {
				return err
			}
			token, err := dependencies.CoderClient.Login(cmd.Context(), email, password)
			if err != nil {
				return err
			}
			if err := dependencies.CredentialStore.Save(cmd.Context(), coderSessionCredentialLabel, token); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), `{"logged_in":true}`)
			return nil
		},
	}
	command.Flags().StringVar(&email, "email", "", "administrator-assigned account email")
	command.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read password from protected stdin")
	_ = command.MarkFlagRequired("email")
	return command
}

func newCoderLogoutCommand(dependencies Dependencies) *cobra.Command {
	return &cobra.Command{Use: "logout", Short: "Revoke this device session", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		token, err := loadCoderToken(cmd.Context(), dependencies)
		if err != nil {
			return err
		}
		if dependencies.CoderClient == nil {
			return errors.New("Coder client is unavailable")
		}
		if err := dependencies.CoderClient.Logout(cmd.Context(), token); err != nil {
			return err
		}
		if err := dependencies.CredentialStore.Delete(cmd.Context(), coderSessionCredentialLabel); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), `{"logged_out":true}`)
		return nil
	}}
}

func newCoderWhoAmICommand(dependencies Dependencies) *cobra.Command {
	return &cobra.Command{Use: "whoami", Short: "Show the current Coder identity", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		token, err := loadCoderToken(cmd.Context(), dependencies)
		if err != nil {
			return err
		}
		if dependencies.CoderClient == nil {
			return errors.New("Coder client is unavailable")
		}
		user, err := dependencies.CoderClient.Me(cmd.Context(), token)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(user)
	}}
}

func newCoderShellCommand(dependencies Dependencies) *cobra.Command {
	return &cobra.Command{Use: "shell <workspace> [-- command...]", Short: "Open a full shell or run a command inside one authorized workspace", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateCoderWorkspaceRef(args[0]); err != nil {
			return err
		}
		coderArgs := []string{"ssh", args[0]}
		if len(args) > 1 {
			coderArgs = append(coderArgs, "--")
			coderArgs = append(coderArgs, args[1:]...)
		}
		return runCoder(cmd, dependencies, coderArgs)
	}}
}

func newCoderPersonalCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "personal", Short: "Manage personal workspaces"}
	root.AddCommand(&cobra.Command{Use: "create <name>", Short: "Create a 50 GiB persistent personal workspace", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateCoderName("workspace", args[0]); err != nil {
			return err
		}
		return runCoder(cmd, dependencies, []string{"create", args[0], "--template", "tcompute-standard", "--use-parameter-defaults", "--yes"})
	}})
	return root
}

func newCoderWorkspaceCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "workspace", Short: "Manage authorized workspaces"}
	root.AddCommand(delegatedCoderCommand(dependencies, "list", "List personal and shared workspaces", cobra.NoArgs, func([]string) ([]string, error) {
		return []string{"list", "--output", "json"}, nil
	}))
	for _, operation := range []string{"start", "stop"} {
		operation := operation
		root.AddCommand(delegatedCoderCommand(dependencies, operation+" <workspace>", operation+" one workspace", cobra.ExactArgs(1), func(args []string) ([]string, error) {
			if err := validateCoderWorkspaceRef(args[0]); err != nil {
				return nil, err
			}
			return []string{operation, args[0], "--yes"}, nil
		}))
	}
	var yes bool
	deleteCommand := delegatedCoderCommand(dependencies, "delete <workspace>", "Permanently delete one workspace", cobra.ExactArgs(1), func(args []string) ([]string, error) {
		if !yes {
			return nil, errors.New("workspace delete requires --yes for the named target")
		}
		if err := validateCoderName("workspace", args[0]); err != nil {
			return nil, err
		}
		return []string{"delete", args[0], "--yes"}, nil
	})
	deleteCommand.Flags().BoolVar(&yes, "yes", false, "confirm permanent deletion of the named workspace")
	root.AddCommand(deleteCommand)
	return root
}

func newCoderOrganizationCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "org", Short: "Manage shared organization workspaces"}
	root.AddCommand(delegatedCoderCommand(dependencies, "list", "List shared organization workspaces", cobra.NoArgs, func([]string) ([]string, error) {
		return []string{"list", "--search", "shared:true", "--output", "json"}, nil
	}))
	root.AddCommand(&cobra.Command{Use: "create <name>", Short: "Create a shared workspace", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateCoderName("workspace", args[0]); err != nil {
			return err
		}
		return runCoder(cmd, dependencies, []string{"create", args[0], "--template", "tcompute-standard", "--use-parameter-defaults", "--yes"})
	}})
	members := &cobra.Command{Use: "member", Short: "Manage access to a shared workspace"}
	members.AddCommand(delegatedCoderCommand(dependencies, "add <workspace> <username>", "Grant workspace use access", cobra.ExactArgs(2), func(args []string) ([]string, error) {
		if err := validateCoderNames(args...); err != nil {
			return nil, err
		}
		return []string{"sharing", "add", args[0], "--user", args[1]}, nil
	}))
	members.AddCommand(&cobra.Command{Use: "remove <workspace> <username>", Short: "Remove access and restart the workspace immediately", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateCoderNames(args...); err != nil {
			return err
		}
		if err := runCoder(cmd, dependencies, []string{"sharing", "remove", args[0], "--user", args[1]}); err != nil {
			return err
		}
		return runCoder(cmd, dependencies, []string{"restart", args[0], "--yes"})
	}})
	members.AddCommand(delegatedCoderCommand(dependencies, "list <workspace>", "List shared members", cobra.ExactArgs(1), func(args []string) ([]string, error) {
		if err := validateCoderName("workspace", args[0]); err != nil {
			return nil, err
		}
		return []string{"sharing", "status", args[0]}, nil
	}))
	root.AddCommand(members)
	return root
}

func newCoderServiceCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{
		Use:   "service",
		Short: "Control the HTTPS workspace service; private is default and public permits anonymous Internet access",
		Long:  "Each workspace exposes port 8000 as an HTTPS application. It is private to the owner by default. Run `tcompute service public <workspace>` only when anonymous Internet access is intended; reverse it with `service private`.",
	}
	for _, definition := range []struct {
		name       string
		visibility string
		short      string
	}{
		{name: "private", visibility: "owner", short: "Require the workspace owner to log in"},
		{name: "authenticated", visibility: "authenticated", short: "Allow logged-in platform users"},
		{name: "public", visibility: "public", short: "Allow anonymous public HTTPS access"},
	} {
		definition := definition
		root.AddCommand(delegatedCoderCommand(dependencies, definition.name+" <workspace>", definition.short, cobra.ExactArgs(1), func(args []string) ([]string, error) {
			if err := validateCoderWorkspaceRef(args[0]); err != nil {
				return nil, err
			}
			return []string{"update", args[0], "--always-prompt", "--use-parameter-defaults", "--parameter", "service_visibility=" + definition.visibility}, nil
		}))
	}
	return root
}

func newCoderSyncCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "sync", Short: "Synchronize files below /home/coder; dry-run is the default"}
	for _, direction := range []string{"push", "pull"} {
		direction := direction
		var apply bool
		command := &cobra.Command{Use: direction + " <workspace> <local-path> <remote-relative-path>", Short: direction + " files using rsync over authenticated Coder SSH", Args: cobra.ExactArgs(3), RunE: func(cmd *cobra.Command, args []string) error {
			if dependencies.CoderSyncer == nil {
				return errors.New("Coder sync is unavailable; reinstall Tashan Compute")
			}
			if err := validateCoderWorkspaceRef(args[0]); err != nil {
				return err
			}
			token, err := loadCoderToken(cmd.Context(), dependencies)
			if err != nil {
				return err
			}
			request := codercli.SyncRequest{Direction: direction, Workspace: args[0], LocalPath: args[1], RemotePath: args[2], Apply: apply}
			return dependencies.CoderSyncer.Sync(cmd.Context(), token, request, cmd.OutOrStdout(), cmd.ErrOrStderr())
		}}
		command.Flags().BoolVar(&apply, "apply", false, "perform the transfer; without this flag rsync only dry-runs")
		root.AddCommand(command)
	}
	return root
}

func newCoderAdminCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "admin", Short: "Platform administrator operations"}
	users := &cobra.Command{Use: "user", Short: "Manage Coder users"}
	var username, email, name string
	var passwordStdin bool
	create := &cobra.Command{Use: "create", Short: "Create an administrator-managed password account", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if dependencies.CoderClient == nil {
			return errors.New("Coder client is unavailable")
		}
		if err := validateCoderName("username", username); err != nil {
			return err
		}
		password, err := readPasswordFromInput(dependencies, passwordStdin)
		if err != nil {
			return err
		}
		token, err := loadCoderToken(cmd.Context(), dependencies)
		if err != nil {
			return err
		}
		user, err := dependencies.CoderClient.CreateUser(cmd.Context(), token, username, email, name, password)
		if err != nil {
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(user)
	}}
	create.Flags().StringVar(&username, "username", "", "new username")
	create.Flags().StringVar(&email, "email", "", "new user email")
	create.Flags().StringVar(&name, "name", "", "display name")
	create.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read initial password from protected stdin")
	_ = create.MarkFlagRequired("username")
	_ = create.MarkFlagRequired("email")
	users.AddCommand(create)

	var resetPasswordStdin bool
	reset := &cobra.Command{Use: "reset-password <username>", Short: "Reset a password through the Coder API", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if dependencies.CoderClient == nil {
			return errors.New("Coder client is unavailable")
		}
		if err := validateCoderName("username", args[0]); err != nil {
			return err
		}
		password, err := readPasswordFromInput(dependencies, resetPasswordStdin)
		if err != nil {
			return err
		}
		token, err := loadCoderToken(cmd.Context(), dependencies)
		if err != nil {
			return err
		}
		if err := dependencies.CoderClient.ResetPassword(cmd.Context(), token, args[0], password); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), `{"password_reset":true}`)
		return nil
	}}
	reset.Flags().BoolVar(&resetPasswordStdin, "password-stdin", false, "read new password from protected stdin")
	users.AddCommand(reset)
	for _, operation := range []string{"suspend", "activate"} {
		operation := operation
		users.AddCommand(delegatedCoderCommand(dependencies, operation+" <username>", operation+" a user", cobra.ExactArgs(1), func(args []string) ([]string, error) {
			if err := validateCoderName("username", args[0]); err != nil {
				return nil, err
			}
			return []string{"users", operation, args[0]}, nil
		}))
	}
	root.AddCommand(users)
	return root
}

func newCoderCapabilityCommand() *cobra.Command {
	root := &cobra.Command{Use: "capability", Short: "Inspect the CLI capability contract"}
	root.AddCommand(&cobra.Command{Use: "list", Short: "Print the machine-readable capability manifest", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		manifest, err := capabilities.Files.ReadFile("manifest.json")
		if err != nil {
			return errors.New("embedded capability manifest is unavailable")
		}
		_, err = cmd.OutOrStdout().Write(append(manifest, '\n'))
		return err
	}})
	return root
}

func newCoderPasswordCommand(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{Use: "password", Short: "Change the current account password"}
	var passwordsStdin bool
	change := &cobra.Command{Use: "change", Short: "Change password, revoke sessions, and require login again", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if dependencies.CoderClient == nil {
			return errors.New("Coder client is unavailable")
		}
		currentPassword, newPassword, err := readPasswordPair(dependencies, passwordsStdin)
		if err != nil {
			return err
		}
		token, err := loadCoderToken(cmd.Context(), dependencies)
		if err != nil {
			return err
		}
		if err := dependencies.CoderClient.ChangePassword(cmd.Context(), token, currentPassword, newPassword); err != nil {
			return err
		}
		if err := dependencies.CredentialStore.Delete(cmd.Context(), coderSessionCredentialLabel); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), `{"password_changed":true,"login_required":true}`)
		return nil
	}}
	change.Flags().BoolVar(&passwordsStdin, "passwords-stdin", false, "read current and new passwords as two protected stdin lines")
	root.AddCommand(change)
	return root
}

type coderArgsBuilder func([]string) ([]string, error)

func delegatedCoderCommand(dependencies Dependencies, use, short string, validator cobra.PositionalArgs, builder coderArgsBuilder) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Args: validator, RunE: func(cmd *cobra.Command, args []string) error {
		coderArgs, err := builder(args)
		if err != nil {
			return err
		}
		return runCoder(cmd, dependencies, coderArgs)
	}}
}

func runCoder(cmd *cobra.Command, dependencies Dependencies, args []string) error {
	if dependencies.CoderRunner == nil {
		return errors.New("Coder CLI is unavailable; reinstall Tashan Compute")
	}
	token, err := loadCoderToken(cmd.Context(), dependencies)
	if err != nil {
		return err
	}
	return dependencies.CoderRunner.Run(cmd.Context(), token, args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func loadCoderToken(ctx context.Context, dependencies Dependencies) (string, error) {
	if dependencies.CredentialStore == nil {
		return "", errors.New("credential store is unavailable")
	}
	token, found, err := dependencies.CredentialStore.Load(ctx, coderSessionCredentialLabel)
	if err != nil {
		return "", err
	}
	if !found || token == "" || strings.ContainsAny(token, "\x00\r\n") {
		return "", errors.New("not logged in; run tcompute login")
	}
	return token, nil
}

func validateCoderNames(values ...string) error {
	for _, value := range values {
		if err := validateCoderName("name", value); err != nil {
			return err
		}
	}
	return nil
}

func validateCoderName(kind, value string) error {
	if !coderNamePattern.MatchString(value) {
		return fmt.Errorf("%s must match %s", kind, coderNamePattern.String())
	}
	return nil
}

func validateCoderWorkspaceRef(value string) error {
	if !coderWorkspaceRefPattern.MatchString(value) {
		return fmt.Errorf("workspace reference must match %s", coderWorkspaceRefPattern.String())
	}
	return nil
}
