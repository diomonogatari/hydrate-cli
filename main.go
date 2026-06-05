// Command hydrate is a quiet, terminal-independent hydration nudge.
package main

import (
	"os"

	"github.com/diomonogatari/hydrate-cli/internal/cli"
)

func main() {
	cli.Assets = assets
	os.Exit(cli.Run(os.Args))
}
