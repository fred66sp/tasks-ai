package cli

import (
	"context"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newShowCommand reads one task by its reference.
func newShowCommand() *cobra.Command {
	return newRefCommand(
		"show <referencia>",
		"Devuelve una tarea",
		"Devuelve en JSON la tarea a la que apunta la referencia, con su identidad interna\n"+
			"y su referencia visible.",
		(*core.Core).Task,
	)
}

// newRefCommand builds a command that works over one reference and answers with
// the JSON of the task it touches. It is the body shared by the commands that
// take exactly one reference.
func newRefCommand(use, short, long string, run func(*core.Core, context.Context, string) (core.Task, error)) *cobra.Command {
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
