package admin

import (
	"context"
	"fmt"
	"io"

	"github.com/TashanGKD/tashan-compute/internal/auth"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Repository        BootstrapRepository
	RepositoryFactory func(context.Context) (BootstrapRepository, func(), error)
	Hasher            auth.PasswordHasher
	IsTerminal        func() bool
	ReadPassword      func(string) (string, error)
	Stdin             io.Reader
}

func NewRoot(dependencies Dependencies) *cobra.Command {
	root := &cobra.Command{
		Use:           "tcompute-admin",
		Short:         "Server-local Tashan Compute administration",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	var username string
	var passwordStdin bool
	bootstrapCommand := &cobra.Command{
		Use:   "bootstrap",
		Short: "Create the first platform administrator using direct database access",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			account, err := bootstrap(context.Background(), dependencies, username, passwordStdin)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "platform administrator created: %s\n", account.ID)
			return nil
		},
	}
	bootstrapCommand.Flags().StringVar(&username, "username", "", "platform administrator username")
	bootstrapCommand.Flags().BoolVar(&passwordStdin, "password-stdin", false, "read the password from stdin")
	_ = bootstrapCommand.MarkFlagRequired("username")
	root.AddCommand(bootstrapCommand)
	return root
}
