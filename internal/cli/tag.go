package cli

import (
	"errors"
	"fmt"
	"strings"
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// newTagsCommand reads the whole catalogue of tags.
func newTagsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tags",
		Short: "Devuelve el catálogo de etiquetas con su uso",
		Long: "Devuelve en JSON el catálogo de etiquetas ordenado por nombre, y cada una con\n" +
			"su nombre, su color, su descripción y el número de tareas que la usan.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			return withProject(command, func(project *project) error {
				catalogue, err := project.core.Tags(command.Context())
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), catalogue)
			})
		},
	}
}

// newTagCommand groups the commands that work over one tag of the catalogue.
func newTagCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "tag",
		Short: "Trabaja con una etiqueta del catálogo",
		Long: "Crea, cambia o borra una etiqueta del catálogo. Las etiquetas viajan por su\n" +
			"nombre, y las tareas guardan las suyas por el número interno de la etiqueta.",
	}

	command.AddCommand(newTagCreateCommand(), newTagEditCommand(), newTagDeleteCommand())

	return command
}

// newTagCreateCommand creates a tag in the catalogue.
func newTagCreateCommand() *cobra.Command {
	var (
		name        string
		color       string
		description string
	)

	command := &cobra.Command{
		Use:   "create",
		Short: "Crea una etiqueta",
		Long: "Crea una etiqueta en el catálogo y devuelve en JSON la etiqueta guardada. El\n" +
			"nombre es obligatorio, y un color que falte es el de por omisión.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			if strings.TrimSpace(name) == "" {
				return &usageError{err: errors.New("falta el nombre de la etiqueta: usa --name")}
			}

			return withProject(command, func(project *project) error {
				created, err := project.core.CreateTag(command.Context(), name, color, description)
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), created)
			})
		},
	}

	command.Flags().StringVar(&name, "name", "", "nombre de la etiqueta (obligatorio)")
	command.Flags().StringVar(&color, "color", "", "color de la etiqueta, uno de "+strings.Join(core.TagColors, ", "))
	command.Flags().StringVar(&description, "description", "", "descripción de la etiqueta")

	return command
}

// newTagEditCommand changes the fields of a tag.
func newTagEditCommand() *cobra.Command {
	var (
		name        string
		color       string
		description string
	)

	command := &cobra.Command{
		Use:   "edit <nombre>",
		Short: "Cambia los campos de una etiqueta",
		Long: "Cambia los campos de una etiqueta y devuelve en JSON la etiqueta tal como\n" +
			"queda. Solo se toca lo que se pide: un campo que no se menciona no cambia, y\n" +
			"renombrar no toca ninguna tarea.",
		Args: usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			changes := core.TagChanges{}

			if command.Flags().Changed("name") {
				changes.Name = &name
			}

			if command.Flags().Changed("color") {
				changes.Color = &color
			}

			if command.Flags().Changed("description") {
				changes.Description = &description
			}

			return withProject(command, func(project *project) error {
				changed, err := project.core.EditTag(command.Context(), args[0], changes)
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), changed)
			})
		},
	}

	command.Flags().StringVar(&name, "name", "", "nuevo nombre de la etiqueta")
	command.Flags().StringVar(&color, "color", "", "nuevo color de la etiqueta, uno de "+strings.Join(core.TagColors, ", "))
	command.Flags().StringVar(&description, "description", "", "nueva descripción, vacía la quita")

	return command
}

// newTagDeleteCommand deletes a tag and its assignments.
func newTagDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <nombre>",
		Short: "Borra una etiqueta",
		Long: "Borra la etiqueta del catálogo, con sus asignaciones, y devuelve en JSON la\n" +
			"etiqueta borrada. Si había tareas que la usaban, avisa de sus referencias por\n" +
			"la salida de errores.",
		Args: usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			return withProject(command, func(project *project) error {
				deleted, users, err := project.core.DeleteTag(command.Context(), args[0])
				if err != nil {
					return err
				}

				if len(users) > 0 {
					_, _ = fmt.Fprintf(command.ErrOrStderr(),
						"aviso: la etiqueta %q la usaban las tareas %s; se han borrado sus asignaciones\n",
						deleted.Name, strings.Join(users, ", "))
				}

				return writeJSON(command.OutOrStdout(), deleted)
			})
		},
	}
}
