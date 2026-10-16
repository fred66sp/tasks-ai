package cli

import (
	"errors"
	"strings"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newCreateCommand creates a task.
func newCreateCommand() *cobra.Command {
	var (
		title       string
		description string
		assignee    string
		notes       string
	)

	command := &cobra.Command{
		Use:   "create",
		Short: "Crea una tarea",
		Long:  "Crea una tarea en el proyecto y devuelve su JSON, con su identidad interna y su referencia.",
		Args:  usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			if strings.TrimSpace(title) == "" {
				return &usageError{err: errors.New("falta el título de la tarea: usa --title")}
			}

			dir, err := workingDir()
			if err != nil {
				return err
			}

			project, err := openProject(command.Context(), dir, command.ErrOrStderr())
			if err != nil {
				return err
			}

			defer func() { _ = project.Close() }()

			task, err := project.core.CreateTask(command.Context(), core.NewTask{
				Title:       title,
				Description: description,
				Assignee:    assignee,
				Notes:       notes,
			})
			if err != nil {
				return err
			}

			return writeJSON(command.OutOrStdout(), task)
		},
	}

	command.Flags().StringVarP(&title, "title", "t", "", "título de la tarea (obligatorio)")
	command.Flags().StringVarP(&description, "description", "d", "", "descripción de la tarea")
	command.Flags().StringVar(&assignee, "assignee", "", "persona responsable de la tarea")
	command.Flags().StringVar(&notes, "notes", "", "notas sueltas de la tarea")

	return command
}
