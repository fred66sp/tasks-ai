// Package cli is the product's command line: it translates commands and flags
// into calls to the core and answers in JSON.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"tasks-ai/internal/core"
	"tasks-ai/internal/install"

	"github.com/spf13/cobra"
)

// Name is how the product's command is called.
const Name = "tasks-ai"

// textRuleNote es la regla del texto en una línea: la llevan las ayudas de los
// comandos que escriben texto, para que --help la nombre.
const textRuleNote = "The text writes no state: the title, the description, the notes " +
	"and the comment body name neither Git nor the state of the task or of the delivery."

// usageError marks an error as the misuse of the command line, which answers
// with its own exit code.
type usageError struct {
	err error
}

func (e *usageError) Error() string {
	return e.err.Error()
}

func (e *usageError) Unwrap() error {
	return e.err
}

// errorBody is how an error crosses the command line's boundary.
type errorBody struct {
	Error string `json:"error"`
}

// New builds the product's command tree. What the product answers goes to out,
// and what it warns or complains about, to errOut: a pipe reads the answer, and
// the warnings never get mixed into it.
func New(out, errOut io.Writer) *cobra.Command {
	root := &cobra.Command{
		Use:   Name,
		Short: "Local task manager for the project",
		Long: "Local task manager for the project.\n\n" +
			"Works on the " + install.DirName + " folder of the project you are in, wherever it is found.\n" +
			"Every operation answers in JSON on the standard output.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.SetOut(out)
	root.SetErr(errOut)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &usageError{err: err}
	})

	root.AddCommand(newInitCommand(), newCreateCommand(), newListCommand(), newShowCommand(), newUpdateCommand(),
		newCloseCommand(), newReopenCommand(), newBlockCommand(), newUnblockCommand(),
		newCommentCommand(), newTagsCommand(), newTagCommand(), newADRCommand(), newServeCommand())

	return root
}

// Run executes the command line with args and returns what the process should
// exit with: nothing when it went well, and the code that says what failed.
func Run(args []string, out, errOut io.Writer) int {
	command := New(out, errOut)
	command.SetArgs(args)

	if err := command.Execute(); err != nil {
		_ = writeJSON(errOut, errorBody{Error: err.Error()})

		// An entry that breaks a rule is the same misuse of the command line as a
		// flag that does not fit: it has its own exit code.
		if _, isMisuse := errors.AsType[*usageError](err); isMisuse || errors.Is(err, core.ErrInvalid) {
			return 2
		}

		return 1
	}

	return 0
}

// Execute runs the product with the arguments the operating system gave it.
func Execute() int {
	return Run(os.Args[1:], os.Stdout, os.Stderr)
}

// usage marks the errors of an argument check as the misuse of the command line.
func usage(check cobra.PositionalArgs) cobra.PositionalArgs {
	return func(command *cobra.Command, args []string) error {
		if err := check(command, args); err != nil {
			return &usageError{err: err}
		}

		return nil
	}
}

// writeJSON writes a value as JSON, which is how the command line always answers.
// Without a place to leave the answer there is no answer: the caller turns the
// write's error into the exit code and into the JSON of the error output.
func writeJSON(out io.Writer, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		_, err = fmt.Fprintf(out, "{\"error\": \"could not prepare the answer: %v\"}\n", err)

		return err
	}

	_, err = fmt.Fprintf(out, "%s\n", raw)

	return err
}
