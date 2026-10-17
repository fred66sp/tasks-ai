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
		Short: "Returns the catalogue of tags with their use",
		Long: "Returns in JSON the catalogue of tags ordered by name, each one with its name,\n" +
			"its color, its description and the number of tasks that use it.",
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
		Short: "Works with one tag of the catalogue",
		Long: "Creates, changes or deletes a tag of the catalogue. Tags travel by their name,\n" +
			"and tasks keep theirs by the internal number of the tag.",
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
		Short: "Creates a tag",
		Long: "Creates a tag in the catalogue and returns the stored tag in JSON. The\n" +
			"name is required, and a color left out is the default one.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			if strings.TrimSpace(name) == "" {
				return &usageError{err: errors.New("the tag name is missing: use --name")}
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

	command.Flags().StringVar(&name, "name", "", "tag name (required)")
	command.Flags().StringVar(&color, "color", "", "tag color, one of "+strings.Join(core.TagColors, ", "))
	command.Flags().StringVar(&description, "description", "", "tag description")

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
		Use:   "edit <name>",
		Short: "Changes the fields of a tag",
		Long: "Changes the fields of a tag and returns the tag as it is left in JSON.\n" +
			"Only what is asked for is touched: a field that is not named does not change,\n" +
			"and renaming does not touch any task.",
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

	command.Flags().StringVar(&name, "name", "", "new tag name")
	command.Flags().StringVar(&color, "color", "", "new tag color, one of "+strings.Join(core.TagColors, ", "))
	command.Flags().StringVar(&description, "description", "", "new description, empty removes it")

	return command
}

// newTagDeleteCommand deletes a tag and its assignments.
func newTagDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Deletes a tag",
		Long: "Deletes the tag from the catalogue, with its assignments, and returns the deleted\n" +
			"tag in JSON. If some tasks used it, it warns about their references on the error\n" +
			"output.",
		Args: usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			return withProject(command, func(project *project) error {
				deleted, users, err := project.core.DeleteTag(command.Context(), args[0])
				if err != nil {
					return err
				}

				if len(users) > 0 {
					_, _ = fmt.Fprintf(command.ErrOrStderr(),
						"warning: the tag %q was used by the tasks %s; their assignments have been deleted\n",
						deleted.Name, strings.Join(users, ", "))
				}

				return writeJSON(command.OutOrStdout(), deleted)
			})
		},
	}
}
