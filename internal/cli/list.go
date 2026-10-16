package cli

import (
	"github.com/spf13/cobra"
)

// newListCommand lists every task of the project.
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Devuelve todas las tareas del proyecto",
		Long: "Devuelve en JSON todas las tareas del proyecto, cada una con su referencia\n" +
			"visible, su estado (pendiente, en curso o hecha), su título y si está bloqueada.",
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
