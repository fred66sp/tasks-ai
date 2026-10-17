package cli

import (
	"context"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newShowCommand reads one task by its reference, with everything around it.
func newShowCommand() *cobra.Command {
	return newRefCommand(
		"show <reference>",
		"Returns a task with what surrounds it",
		"Returns in JSON the task the reference points at, with its internal identity\n"+
			"and its visible reference, its children, its blockers, its dependents and its\n"+
			"comments.",
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
