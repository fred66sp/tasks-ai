package cli

import (
	"github.com/spf13/cobra"
)

// newListCommand lists every task of the project.
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Returns every task of the project",
		Long: "Returns in JSON every task of the project, each one with its visible reference,\n" +
			"its status (open, in progress or done), its title and whether it is blocked.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			return withProject(command, func(project *project) error {
				tasks, err := project.core.Tasks(command.Context())
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), tasks)
			})
		},
	}
}
