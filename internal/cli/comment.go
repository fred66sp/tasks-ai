package cli

import (
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newCommentCommand appends a comment to a task.
func newCommentCommand() *cobra.Command {
	return newFlagRefCommand(
		"comment <referencia>",
		"Añade un comentario a una tarea",
		"Añade un comentario a la tarea y devuelve en JSON la tarea tal como está. Los\n"+
			"comentarios solo se añaden: no hay manera de editar ni de borrar uno.",
		"body",
		"cuerpo del comentario (obligatorio)",
		(*core.Core).AddComment,
	)
}
