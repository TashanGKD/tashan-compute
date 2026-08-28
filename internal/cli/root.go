package cli

import (
	"github.com/TashanGKD/tashan-compute/internal/buildinfo"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	NetworkProbe func()
}

func NewRoot(_ Dependencies) *cobra.Command {
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
	return cmd
}
