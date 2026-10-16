package cli

import (
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newReopenCommand returns a closed task to open.
func newReopenCommand() *cobra.Command {
	return newRefCommand(
		"reopen <referencia>",
		"Reabre una tarea cerrada",
		"Reabre una tarea cerrada: devuelve su estado a open, le quita el motivo de cierre\n"+
			"y devuelve en JSON la tarea tal como queda.",
		(*core.Core).ReopenTask,
	)
}
