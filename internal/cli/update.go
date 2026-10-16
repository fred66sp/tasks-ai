package cli

import (
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newUpdateCommand changes the fields of a task.
func newUpdateCommand() *cobra.Command {
	var (
		title       string
		description string
		notes       string
		assignee    string
		priority    int
	)

	command := &cobra.Command{
		Use:   "update <referencia>",
		Short: "Cambia los campos de una tarea",
		Long: "Cambia los campos de una tarea y devuelve en JSON la tarea tal como queda.\n" +
			"Solo se toca lo que se pide: un campo que no se menciona no cambia, y un texto\n" +
			"dado vacío deja ese campo sin valor.",
		Args: usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			changes := core.Changes{}

			if command.Flags().Changed("title") {
				changes.Title = &title
			}

			if command.Flags().Changed("description") {
				changes.Description = &description
			}

			if command.Flags().Changed("notes") {
				changes.Notes = &notes
			}

			if command.Flags().Changed("assignee") {
				changes.Assignee = &assignee
			}

			if command.Flags().Changed("priority") {
				changes.Priority = &priority
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

			task, err := project.core.UpdateTask(command.Context(), args[0], changes)
			if err != nil {
				return err
			}

			return writeJSON(command.OutOrStdout(), task)
		},
	}

	command.Flags().StringVarP(&title, "title", "t", "", "nuevo título de la tarea")
	command.Flags().StringVarP(&description, "description", "d", "", "nueva descripción, vacía la quita")
	command.Flags().StringVar(&notes, "notes", "", "nuevas notas, vacías las quitan")
	command.Flags().StringVar(&assignee, "assignee", "", "nueva persona responsable, vacía la quita")
	command.Flags().IntVar(&priority, "priority", core.DefaultPriority, "nueva prioridad, de 0 a 3")

	return command
}
