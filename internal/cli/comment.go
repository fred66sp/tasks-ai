package cli

import (
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newCommentCommand appends a comment to a task.
func newCommentCommand() *cobra.Command {
	return newFlagRefCommand(
		"comment <reference>",
		"Adds a comment to a task",
		"Adds a comment to the task and returns the task as it is in JSON. Comments are\n"+
			"only added: there is no way to edit or delete one.",
		"body",
		"comment body (required)",
		(*core.Core).AddComment,
	)
}
