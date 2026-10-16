package cli

import (
	"context"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newShowCommand reads one task by its reference, with everything around it.
func newShowCommand() *cobra.Command {
	return newRefCommand(
		"show <referencia>",
		"Devuelve una tarea con lo que la rodea",
		"Devuelve en JSON la tarea a la que apunta la referencia, con su identidad interna\n"+
			"y su referencia visible, sus hijas, sus bloqueantes, sus dependientes y sus\n"+
			"comentarios.",
		(*core.Core).TaskDetail,
	)
}

// newRefCommand builds a command that works over one reference and answers with
// the JSON of what the core returns: the plain task, or its detail.
func newRefCommand[T any](use, short, long string, run func(*core.Core, context.Context, string) (T, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			return withProject(command, func(project *project) error {
				task, err := run(project.core, command.Context(), args[0])
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), task)
			})
		},
	}
}
