package cli

import (
	"context"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newBlockCommand leaves a task blocked by another.
func newBlockCommand() *cobra.Command {
	return newPairCommand(
		"block <reference> <blocker-reference>",
		"Leaves a task blocked by another",
		"Leaves the first task blocked by the second: while the blocker is not\n"+
			"done, the first one shows as blocked. A reference that does not resolve is refused\n"+
			"and a task cannot be blocked by itself.",
		(*core.Core).BlockTask,
	)
}

// newUnblockCommand removes the block between two tasks.
func newUnblockCommand() *cobra.Command {
	return newPairCommand(
		"unblock <reference> <blocker-reference>",
		"Removes the block between two tasks",
		"Removes the block that leaves the first task blocked by the second and returns in\n"+
			"JSON the task as it is left, which stops showing as blocked if it had no other\n"+
			"blockers left.",
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
