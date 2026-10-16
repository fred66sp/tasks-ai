package cli

import (
	"context"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newBlockCommand leaves a task blocked by another.
func newBlockCommand() *cobra.Command {
	return newPairCommand(
		"block <referencia> <referencia-bloqueante>",
		"Deja una tarea bloqueada por otra",
		"Deja la primera tarea bloqueada por la segunda: mientras la bloqueante no esté\n"+
			"done, la primera aparece bloqueada. Una referencia que no resuelve se rechaza y\n"+
			"una tarea no puede bloquearse por sí misma.",
		(*core.Core).BlockTask,
	)
}

// newUnblockCommand removes the block between two tasks.
func newUnblockCommand() *cobra.Command {
	return newPairCommand(
		"unblock <referencia> <referencia-bloqueante>",
		"Retira el bloqueo entre dos tareas",
		"Retira el bloqueo que deja la primera tarea bloqueada por la segunda y devuelve en\n"+
			"JSON la tarea tal como queda, que deja de aparecer bloqueada si no le quedaban\n"+
			"más bloqueantes.",
		(*core.Core).UnblockTask,
	)
}

// newPairCommand builds a command that works over two references and answers
// with the JSON of the task the first one points at. It is the body shared by
// the commands that take exactly two references.
func newPairCommand(use, short, long string, run func(*core.Core, context.Context, string, string) (core.Task, error)) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  usage(cobra.ExactArgs(2)),
		RunE: func(command *cobra.Command, args []string) error {
			return withProject(command, func(project *project) error {
				task, err := run(project.core, command.Context(), args[0], args[1])
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), task)
			})
		},
	}
}
