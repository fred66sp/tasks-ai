// Command tasks-ai is the product's only binary: the command line and, when it
// is asked for, the server of the interface.
package main

import (
	"os"
	"tasks-ai/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
