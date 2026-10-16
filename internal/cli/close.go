package cli

import (
	"github.com/spf13/cobra"
)

// newCloseCommand closes a task with a reason.
func newCloseCommand() *cobra.Command {
	var reason string

	command := &cobra.Command{
		Use:   "close <referencia>",
		Short: "Cierra una tarea con un motivo",
		Long: "Cierra una tarea: deja su estado en done, guarda el motivo y devuelve en JSON la\n" +
			"tarea tal como queda. El motivo es obligatorio y debe ser uno de los conocidos.",
		Args: usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			// El motivo ausente o desconocido lo rechaza el núcleo: la lista de
			// motivos vive en un solo sitio y el error sale con el código de uso.
			return withProject(command, func(project *project) error {
				task, err := project.core.CloseTask(command.Context(), args[0], reason)
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), task)
			})
		},
	}

	command.Flags().StringVar(&reason, "reason", "", "motivo del cierre: done, duplicate, absorbed, stale o superseded (obligatorio)")

	return command
}
