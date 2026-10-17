package cli

import (
	"tasks-ai/internal/install"

	"github.com/spf13/cobra"
)

// newInitCommand installs the product in the project.
func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Installs the product in the project",
		Long: "Installs the product in the project: it leaves its folder with the database, the\n" +
			"settings and the instructions, and returns the text to add outside to point at them.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			dir, err := workingDir()
			if err != nil {
				return err
			}

			result, err := install.Run(command.Context(), install.Root(dir))
			if err != nil {
				return err
			}

			return writeJSON(command.OutOrStdout(), result)
		},
	}
}
