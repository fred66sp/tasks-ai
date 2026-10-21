package cli

import (
	"fmt"
	"strconv"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newCommentCommand appends a comment to a task, and carries the subcommands
// that read and edit the comments of the task.
func newCommentCommand() *cobra.Command {
	command := newFlagRefCommand(
		"comment <reference>",
		"Adds a comment to a task",
		"Adds a comment to the task and returns the task as it is in JSON. The body is\n"+
			"required and is stored without the spaces at its sides. Use comment list to\n"+
			"read the comments of the task and comment edit to change the body of one of\n"+
			"them.\n"+
			textRuleNote,
		"body",
		"comment body (required)",
		(*core.Core).AddComment,
	)

	command.AddCommand(newCommentListCommand(), newCommentEditCommand())

	return command
}

// newCommentListCommand reads the comments of a task, from the first to the
// last, each one with its identity.
func newCommentListCommand() *cobra.Command {
	return newRefCommand(
		"list <reference>",
		"Returns the comments of a task",
		"Returns in JSON the comments of the task, from the first to the last, each one\n"+
			"with its identity, its body and the moment it was added at.",
		(*core.Core).Comments,
	)
}

// newCommentEditCommand changes the body of one comment of the task. The number
// is understood inside the task, and the identity and the date of the comment
// are kept.
func newCommentEditCommand() *cobra.Command {
	var body string

	command := &cobra.Command{
		Use:   "edit <reference> <id>",
		Short: "Changes the body of a comment",
		Long: "Changes the body of the comment of the task that carries the number given, and\n" +
			"returns the task as it is left in JSON. The identity and the creation date of\n" +
			"the comment are kept; a body is required.\n" +
			textRuleNote,
		Args: usage(cobra.ExactArgs(2)),
		RunE: func(command *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return &usageError{err: fmt.Errorf("the comment id %q is not a number", args[1])}
			}

			return withProject(command, func(project *project) error {
				task, err := project.core.EditComment(command.Context(), args[0], id, body)
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), task)
			})
		},
	}

	command.Flags().StringVar(&body, "body", "", "new comment body (required)")

	return command
}
