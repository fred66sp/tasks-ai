package cli

import "github.com/spf13/cobra"

// newShowCommand reads one task by its reference.
func newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <referencia>",
		Short: "Devuelve una tarea",
		Long: "Devuelve en JSON la tarea a la que apunta la referencia, con su identidad interna\n" +
			"y su referencia visible.",
		Args: usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			dir, err := workingDir()
			if err != nil {
				return err
			}

			project, err := openProject(command.Context(), dir, command.ErrOrStderr())
			if err != nil {
				return err
			}

			defer func() { _ = project.Close() }()

			task, err := project.core.Task(command.Context(), args[0])
			if err != nil {
				return err
			}

			writeJSON(command.OutOrStdout(), task)

			return nil
		},
	}
}
