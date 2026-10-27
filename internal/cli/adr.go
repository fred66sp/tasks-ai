package cli

import (
	"errors"
	"strings"
	"tasks-ai/internal/core"
	"tasks-ai/internal/store"

	"github.com/spf13/cobra"
)

// newADRCommand groups the operations of the decisions of the project.
func newADRCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "adr",
		Short: "Reads and writes the decisions of the project",
		Long: "Reads and writes the ADR of the project: the decisions that outlive the work.\n" +
			"Each ADR has a number, a reference ADR-NNN and a body by sections; once it leaves\n" +
			"proposed, the body is frozen and only more information accepts additions.\n" +
			"docs/decisions/ is a generated export of this register: see adr export.",
	}

	command.AddCommand(
		newADRListCommand(),
		newADRShowCommand(),
		newADRCreateCommand(),
		newADREditCommand(),
		newADRImportCommand(),
		newADRExportCommand(),
	)

	return command
}

// newADRListCommand lists the ADR with their context, without the rest of the
// body.
func newADRListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lists the ADRs",
		Long:  "Lists the ADR of the project by number, each one with its reference, its state, its day\nand its context: the body of the decision is read with adr show.",
		Args:  usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			return withProject(command, func(opened *project) error {
				adrs, err := opened.core.ADRs(command.Context(), "")
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), adrs)
			})
		},
	}
}

// newADRShowCommand returns one ADR with its whole body.
func newADRShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <number>",
		Short: "Shows one ADR",
		Long: "Shows the ADR the number or the reference points at, with its whole body and, " +
			"when it is superseded, the ADR that replaces it.",
		Args: usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			return withProject(command, func(opened *project) error {
				adr, err := opened.core.ADR(command.Context(), args[0])
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), adr)
			})
		},
	}
}

// newADRCreateCommand creates an ADR.
func newADRCreateCommand() *cobra.Command {
	var (
		title        string
		deciders     string
		status       string
		context      string
		decision     string
		consequences string
		alternatives string
	)

	command := &cobra.Command{
		Use:   "create",
		Short: "Creates an ADR",
		Long: "Creates an ADR and returns its JSON, with its number and its reference.\n" +
			"The ADR is born proposed: the body that is missing is written with adr edit,\n" +
			"and the status changes when the decision is taken.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			return withProject(command, func(opened *project) error {
				adr, err := opened.core.CreateADR(command.Context(), core.NewADR{
					Title:        title,
					Deciders:     deciders,
					Status:       status,
					Context:      context,
					Decision:     decision,
					Consequences: consequences,
					Alternatives: alternatives,
				})
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), adr)
			})
		},
	}

	command.Flags().StringVarP(&title, "title", "t", "", "ADR title (required)")
	command.Flags().StringVar(&deciders, "deciders", "", "people who own the decision (required)")
	command.Flags().StringVar(&status, "status", store.ADRProposed, "state of the ADR: proposed, accepted, rejected, deprecated or superseded")
	command.Flags().StringVar(&context, "context", "", "context and problem of the decision")
	command.Flags().StringVar(&decision, "decision", "", "what is decided, with its scope")
	command.Flags().StringVar(&consequences, "consequences", "", "what the decision makes easier and harder")
	command.Flags().StringVar(&alternatives, "alternatives", "", "the alternatives and why they were rejected")

	return command
}

// newADREditCommand changes what the command names.
func newADREditCommand() *cobra.Command {
	var (
		title        string
		status       string
		supersededBy string
		context      string
		decision     string
		consequences string
		alternatives string
		note         string
	)

	command := &cobra.Command{
		Use:   "edit <number>",
		Short: "Changes an ADR",
		Long: "Changes only what the flags name and returns the ADR as it is left.\n" +
			"The body is frozen once the ADR leaves proposed: from then on only --note adds to\n" +
			"more information, and a change of decision is a new ADR that supersedes this one.",
		Args: usage(cobra.ExactArgs(1)),
		RunE: func(command *cobra.Command, args []string) error {
			changes := core.ADRChanges{}

			if command.Flags().Changed("title") {
				changes.Title = &title
			}

			if command.Flags().Changed("status") {
				changes.Status = &status
			}

			if command.Flags().Changed("context") {
				changes.Context = &context
			}

			if command.Flags().Changed("decision") {
				changes.Decision = &decision
			}

			if command.Flags().Changed("consequences") {
				changes.Consequences = &consequences
			}

			if command.Flags().Changed("alternatives") {
				changes.Alternatives = &alternatives
			}

			if command.Flags().Changed("note") {
				changes.Note = &note
			}

			if command.Flags().Changed("superseded-by") {
				number, err := core.ParseADRRef(supersededBy)
				if err != nil {
					return err
				}

				changes.SupersededBy = &number
			}

			return withProject(command, func(opened *project) error {
				adr, err := opened.core.UpdateADR(command.Context(), args[0], changes)
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), adr)
			})
		},
	}

	command.Flags().StringVarP(&title, "title", "t", "", "new title of the ADR")
	command.Flags().StringVar(&status, "status", "", "new state: proposed, accepted, rejected, deprecated or superseded")
	command.Flags().StringVar(&supersededBy, "superseded-by", "", "ADR that replaces this one, by its number or ADR-NNN; name it back from the substitute's more information")
	command.Flags().StringVar(&context, "context", "", "new context and problem")
	command.Flags().StringVar(&decision, "decision", "", "new decision")
	command.Flags().StringVar(&consequences, "consequences", "", "new consequences")
	command.Flags().StringVar(&alternatives, "alternatives", "", "new alternatives")
	command.Flags().StringVar(&note, "note", "", "addition to more information, dated today")

	return command
}

// newADRImportCommand reads the ADR of a directory of markdown files.
func newADRImportCommand() *cobra.Command {
	var dir string

	command := &cobra.Command{
		Use:   "import",
		Short: "Imports the ADRs of a directory of files",
		Long: "Reads the markdown files of the directory and stores the ADR that are not in the\n" +
			"database yet, keeping their number, state, day, deciders, sections and the ADR that\n" +
			"supersedes each one. It is idempotent by number: repeating it neither duplicates an\n" +
			"ADR nor rewrites it, and the index of the directory is left alone.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			if err := adrDir(dir); err != nil {
				return err
			}

			entries, err := parseADRDir(dir)
			if err != nil {
				return err
			}

			return withProject(command, func(opened *project) error {
				result, err := opened.core.ImportADRs(command.Context(), entries)
				if err != nil {
					return err
				}

				return writeJSON(command.OutOrStdout(), result)
			})
		},
	}

	command.Flags().StringVar(&dir, "dir", "", "directory of markdown ADR files (required)")

	return command
}

// adrDir refuses an import without a directory to read.
func adrDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return &usageError{err: errors.New("the directory of ADR files is missing: use --dir")}
	}

	return nil
}
