package cli

import (
	"context"
	"io"

	"github.com/TashanGKD/tashan-compute/internal/buildinfo"
	"github.com/TashanGKD/tashan-compute/internal/codercli"
	"github.com/TashanGKD/tashan-compute/internal/credentials"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	NetworkProbe    func()
	LoginClient     LoginClient
	APIClient       APIClient
	CredentialStore credentials.Store
	Stdin           io.Reader
	IsTerminal      func() bool
	ReadPassword    func(string) (string, error)
	CoderClient     CoderClient
	CoderRunner     CoderCommandRunner
	CoderSyncer     CoderSyncer
}

type CoderClient interface {
	Login(context.Context, string, string) (string, error)
	Me(context.Context, string) (codercli.User, error)
	Logout(context.Context, string) error
	ChangePassword(context.Context, string, string, string) error
	CreateUser(context.Context, string, string, string, string, string) (codercli.User, error)
	ResetPassword(context.Context, string, string, string) error
}

type CoderCommandRunner interface {
	Run(context.Context, string, []string, io.Reader, io.Writer, io.Writer) error
}

type CoderSyncer interface {
	Sync(context.Context, string, codercli.SyncRequest, io.Writer, io.Writer) error
}

func NewRoot(dependencies Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "tcompute",
		Short:         "Tashan Compute CLI",
		Long:          "Tashan Compute manages authorized personal and organization compute spaces.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.Version = buildinfo.Version
	cmd.SetVersionTemplate("{{.Version}}\n")
	if dependencies.CoderClient != nil || dependencies.CoderRunner != nil {
		addCoderCommands(cmd, dependencies)
	} else {
		cmd.AddCommand(newAuthCommand(dependencies))
		cmd.AddCommand(newHealthCommand(dependencies))
		cmd.AddCommand(newCapabilityCommand(dependencies))
		cmd.AddCommand(newDeviceCommand(dependencies))
		cmd.AddCommand(newAdminCommand(dependencies))
		cmd.AddCommand(newOrganizationCommand(dependencies))
		cmd.AddCommand(newAuditCommand(dependencies))
	}
	return cmd
}
