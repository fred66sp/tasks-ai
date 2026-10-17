package cli

import (
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newReopenCommand returns a closed task to open.
func newReopenCommand() *cobra.Command {
	return newRefCommand(
		"reopen <reference>",
		"Reopens a closed task",
		"Reopens a closed task: it returns its status to open, clears the close reason\n"+
			"and returns the task as it is left in JSON.",
		(*core.Core).ReopenTask,
	)
}
