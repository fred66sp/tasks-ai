package cli

import (
	"tasks-ai/internal/install"

	"github.com/spf13/cobra"
)

// newInitCommand installs the product in the project.
func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Instala el producto en el proyecto",
		Long: "Instala el producto en el proyecto: deja su carpeta con la base, los ajustes y las\n" +
			"instrucciones, y devuelve el texto que hay que añadir fuera para apuntar a ellas.",
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

			writeJSON(command.OutOrStdout(), result)

			return nil
		},
	}
}
