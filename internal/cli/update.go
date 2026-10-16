package cli

import (
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"

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
		tags        string
		status      string
	)

	command := &cobra.Command{
		Use:   "update <referencia>",
		Short: "Cambia los campos de una tarea",
		Long: "Cambia los campos de una tarea y devuelve en JSON la tarea tal como queda.\n" +
			"Solo se toca lo que se pide: un campo que no se menciona no cambia, y un texto\n" +
			"dado vacío deja ese campo sin valor.\n" +
			"Con --status se cambia el estado: open (pendiente) o in_progress (en curso).\n" +
			"Cerrar no va por aquí: usa close, con su motivo.",
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

			if command.Flags().Changed("tags") {
				names := splitTags(tags)
				changes.Tags = &names
			}

			if command.Flags().Changed("status") {
				changes.Status = &status
			}

			dir, err := workingDir()
			if err != nil {
				return err
			}

			project, err := openProject(command.Context(), dir, command.ErrOrStderr(), store.OriginCLI)
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
	command.Flags().StringVar(&tags, "tags", "", "nuevas etiquetas de la tarea, separadas por comas; una lista vacía las quita todas")
	command.Flags().StringVar(&status, "status", "", "nuevo estado de la tarea: open (pendiente) o in_progress (en curso); para cerrar usa close, con su motivo")

	return command
}

// splitTags parts the list of names --tags carries, separated by commas and
// with spare space around each name. The empty pieces fall out: a list left
// empty leaves the task without any tag.
func splitTags(list string) []string {
	names := []string{}

	for name := range strings.SplitSeq(list, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}

	return names
}
