package cli

import (
	"errors"
	"fmt"
	"tasks-ai/internal/install"

	"github.com/spf13/cobra"
)

// newUpgradeCommand brings the installation of the current project up to this
// binary, without downloading anything.
func newUpgradeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrades the project's installation to this binary's version",
		Long: "Upgrades the installation of the project you are in to the version of this\n" +
			"binary: it copies the binary, refreshes the label contract and the instructions,\n" +
			"removes the host references and the extension folders of the retired integrations\n" +
			"and records the version last. It refuses a downgrade and answers in JSON with what\n" +
			"it did and what is left pending.",
		Args: usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			dir, err := workingDir()
			if err != nil {
				return err
			}

			report, err := install.Upgrade(dir)
			if err != nil {
				if errors.Is(err, install.ErrUpgradeRefused) {
					return &usageError{err: err}
				}

				return err
			}

			if err := writeJSON(command.OutOrStdout(), report); err != nil {
				return err
			}

			if report.Instructions == "pending" {
				_, _ = fmt.Fprintf(command.ErrOrStderr(),
					"warning: the instructions of the project are not the ones the product writes: review them by hand\n")
			}

			if report.Contract == "pending" {
				_, _ = fmt.Fprintf(command.ErrOrStderr(),
					"warning: the contract of the labels was edited by the project: the product leaves it as it is\n")
			}

			if report.Version == "" {
				for _, pending := range report.Pending {
					_, _ = fmt.Fprintf(command.ErrOrStderr(), "warning: %s\n", pending)
				}

				return exitCodeError(2)
			}

			return nil
		},
	}
}
