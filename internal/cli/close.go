package cli

import (
	"context"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newCloseCommand closes a task with a reason.
func newCloseCommand() *cobra.Command {
	return newFlagRefCommand(
		"close <reference>",
		"Closes a task with a reason",
		"Closes a task: it leaves its status in done, records the reason and returns the\n"+
			"task as it is left in JSON. The reason is required and must be one of the known ones.",
		"reason",
		"close reason: done, duplicate, absorbed, stale or superseded (required)",
		(*core.Core).CloseTask,
	)
}

// newFlagRefCommand builds a command that works over one reference and one
// flag, hands both to the core and answers with the JSON of the task. It is the
// body shared by the commands that take exactly one reference and one flag.
func newFlagRefCommand(use, short, long, flag, flagHelp string, run func(*core.Core, context.Context, string, string) (core.Task, error)) *cobra.Command {
	var value string

	command := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			// La regla sobre el valor de la bandera vive en el núcleo: la
			// comprobación queda en un solo sitio y el error sale con el
			// código de uso.
			return withProject(command, func(project *project) error {
				task, err := run(project.core, command.Context(), args[0], value)
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), task)
			})
		},
	}

	command.Flags().StringVar(&value, flag, "", flagHelp)

	return command
}
