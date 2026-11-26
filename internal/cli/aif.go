package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"syscall"
	"tasks-ai/internal/aif"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"

	interaction "github.com/fred66sp/agent-interaction-forms"
	"github.com/spf13/cobra"
)

// aifUsage is the help of the profile: the command reads one request and
// answers one object, and nothing it prints wants to be read by a person.
const aifUsage = "Serves the AIF 0.2 CLI profile.\n\n" +
	"Reads one AIF request object from the standard input, answers exactly one JSON\n" +
	"object on the standard output and writes its diagnostics on the standard error.\n" +
	"Every invocation is independent: the continuation sends the whole corrected\n" +
	"request again. The exit code is the one docs/contracts.md §7 fixes for the\n" +
	"result: 0 executed or discovery, 2 input_needed or invalid, 3 unknown_operation,\n" +
	"unsupported_version or malformed_request, 4 denied, 1 failed."

// newAIFCommand serves the AIF 0.2 CLI profile: the operations of
// docs/contracts.md §7 over the core of the project the command is called from.
func newAIFCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "aif",
		Short: "Serves the AIF 0.2 CLI profile",
		Long:  aifUsage,
		Args:  usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			return serveAIF(command)
		},
	}
}

// serveAIF opens the project, serves one request and returns the exit code the
// profile fixes for its result.
func serveAIF(command *cobra.Command) error {
	dir, err := workingDir()
	if err != nil {
		return failedAIF(command, err)
	}

	opened, err := openProject(command.Context(), dir, command.ErrOrStderr(), store.OriginCLI)
	if err != nil {
		return failedAIF(command, err)
	}

	defer func() { _ = opened.Close() }()

	if code := runAIF(command.InOrStdin(), command.OutOrStdout(), command.ErrOrStderr(), opened.core); code != 0 {
		return exitCodeError(code)
	}

	return nil
}

// failedAIF answers a failure of the command itself — no project, no standard
// input — with the profile's failed response, so a consumer always reads one.
func failedAIF(command *cobra.Command, err error) error {
	response := &interaction.Response{
		AIF:     interaction.Version,
		Result:  interaction.ResultFailed,
		Message: err.Error(),
	}

	if code := answer(command.OutOrStdout(), command.ErrOrStderr(), response); code != 0 {
		return exitCodeError(code)
	}

	return nil
}

// runAIF is the profile's channel: one request from stdin, one JSON object on
// stdout, diagnostics in prose on stderr, and the code its result fixes.
func runAIF(stdin io.Reader, stdout, stderr io.Writer, apply *core.Core) int {
	raw, err := io.ReadAll(io.LimitReader(stdin, interaction.MaxInputBytes+1))
	if err != nil {
		return answer(stdout, stderr, &interaction.Response{
			AIF:     interaction.Version,
			Result:  interaction.ResultFailed,
			Message: "could not read the request",
			Code:    "stdin_unavailable",
		})
	}

	return answer(stdout, stderr, aif.New(apply).Handle(raw))
}

// answer writes the profile's one object and returns its exit code. A write
// that nobody reads is not a failure of the product, and a stdout that cannot
// take the answer is: the diagnostic goes to stderr, never into the answer.
func answer(stdout, stderr io.Writer, response *interaction.Response) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(response); err != nil {
		if errors.Is(err, syscall.EPIPE) {
			return 0
		}

		_, _ = fmt.Fprintf(stderr, "%s: could not write the answer: %v\n", Name, err)

		return 1
	}

	return exitCodeOf(response)
}

// exitCodeOf maps the result to the exit code of §7. Discovery is not an
// operation result, and it exits with the same code as executed.
func exitCodeOf(response *interaction.Response) int {
	if response.Discovery != nil {
		return 0
	}

	switch response.Result {
	case interaction.ResultExecuted:
		return 0
	case interaction.ResultInputNeeded, interaction.ResultInvalid:
		return 2
	case interaction.ResultDenied:
		return 4
	case interaction.ResultFailed:
		return 1
	default:
		return 3
	}
}
