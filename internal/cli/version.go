package cli

import (
	"tasks-ai/internal/core"

	"github.com/spf13/cobra"
)

// versionBody is the version of the product as it crosses the command line.
type versionBody struct {
	Version string `json:"version"`
}

// newVersionCommand answers the version of the product.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the version of the product",
		Long:  "Prints the version of the product as a JSON object on the standard output.",
		Args:  usage(cobra.NoArgs),
		RunE: func(command *cobra.Command, _ []string) error {
			return writeJSON(command.OutOrStdout(), versionBody{Version: core.Version})
		},
	}
}
