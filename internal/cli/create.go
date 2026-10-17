package cli

import (
	"errors"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"

	"github.com/spf13/cobra"
)

// newCreateCommand creates a task.
func newCreateCommand() *cobra.Command {
	var (
		title       string
		description string
		assignee    string
		notes       string
		priority    int
		parent      string
	)

	command := &cobra.Command{
		Use:   "create",
		Short: "Creates a task",
		Long:  "Creates a task in the project and returns its JSON, with its internal identity and its reference.",
		Args:  usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			if strings.TrimSpace(title) == "" {
				return &usageError{err: errors.New("the task title is missing: use --title")}
			}

			newTask := core.NewTask{
				Title:       title,
				Description: description,
				Assignee:    assignee,
				Notes:       notes,
			}

			if command.Flags().Changed("priority") {
				newTask.Priority = &priority
			}

			if command.Flags().Changed("parent") {
				newTask.Parent = parent
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

			task, err := project.core.CreateTask(command.Context(), newTask)
			if err != nil {
				return err
			}

			return writeJSON(command.OutOrStdout(), task)
		},
	}

	command.Flags().StringVarP(&title, "title", "t", "", "task title (required)")
	command.Flags().StringVarP(&description, "description", "d", "", "task description")
	command.Flags().StringVar(&assignee, "assignee", "", "person responsible for the task")
	command.Flags().StringVar(&notes, "notes", "", "loose notes of the task")
	command.Flags().IntVar(&priority, "priority", core.DefaultPriority, "task priority, from 0 to 3")
	command.Flags().StringVar(&parent, "parent", "", "reference of the parent task, to create a subtask")

	return command
}
