package apicmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
)

type Runtime interface {
	Serve(context.Context) error
}

type Dependencies struct {
	Build func(context.Context) (Runtime, error)
}

func NewRoot(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{
		Use:           "tcompute-api",
		Short:         "Tashan Compute API server",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	root.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Serve the configured API",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dependencies.Build == nil {
				return errors.New("API runtime builder is unavailable")
			}
			runtime, err := dependencies.Build(cmd.Context())
			if err != nil {
				return err
			}
			return runtime.Serve(cmd.Context())
		},
	})
	return root
}
