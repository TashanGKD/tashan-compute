package cli

import (
	"io"

	"github.com/TashanGKD/tashan-compute/internal/buildinfo"
	"github.com/TashanGKD/tashan-compute/internal/credentials"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	NetworkProbe    func()
	LoginClient     LoginClient
	CredentialStore credentials.Store
	Stdin           io.Reader
	IsTerminal      func() bool
	ReadPassword    func(string) (string, error)
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
	cmd.AddCommand(newAuthCommand(dependencies))
	return cmd
}
