package cli

import (
	"strings"
	"tasks-ai/internal/install"
	"tasks-ai/internal/settings"

	"github.com/spf13/cobra"
)

// newInitCommand installs the product in the project.
func newInitCommand() *cobra.Command {
	var prefix string

	command := &cobra.Command{
		Use:   "init",
		Short: "Installs the product in the project",
		Long: "Installs the product in the project: it leaves its folder with the database, the\n" +
			"settings and the instructions, and returns the text to add outside to point at them.\n" +
			"With --prefix the references of the project are composed with that prefix instead\n" +
			"of tk, and the instructions are written with it.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			prefix = strings.TrimSpace(prefix)

			if prefix != "" {
				if err := settings.ValidatePrefix(prefix); err != nil {
					return &usageError{err: err}
				}
			}

			dir, err := workingDir()
			if err != nil {
				return err
			}

			result, err := install.Run(command.Context(), install.Root(dir), prefix)
			if err != nil {
				return err
			}

			return writeJSON(command.OutOrStdout(), result)
		},
	}

	command.Flags().StringVar(&prefix, "prefix", "", "the prefix of the project's references, instead of tk")

	return command
}
