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
		Use:   "update <reference>",
		Short: "Changes the fields of a task",
		Long: "Changes the fields of a task and returns the task as it is left in JSON.\n" +
			"Only what is asked for is touched: a field that is not named does not change, and\n" +
			"an empty text leaves that field without a value.\n" +
			"With --status the status changes: open or in_progress.\n" +
			"Closing does not go through here: use close, with its reason.",
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

	command.Flags().StringVarP(&title, "title", "t", "", "new task title")
	command.Flags().StringVarP(&description, "description", "d", "", "new description, empty removes it")
	command.Flags().StringVar(&notes, "notes", "", "new notes, empty removes them")
	command.Flags().StringVar(&assignee, "assignee", "", "new person responsible, empty removes it")
	command.Flags().IntVar(&priority, "priority", core.DefaultPriority, "new priority, from 0 to 3")
	command.Flags().StringVar(&tags, "tags", "", "new task tags, separated by commas; an empty list removes all of them")
	command.Flags().StringVar(&status, "status", "", "new task status: open or in_progress; to close use close, with its reason")

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
