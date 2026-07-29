// Command git-snag is a TUI for cleaning up git worktrees and their branches
// across repositories found under the current directory.
package main

import (
	"os"

	"github.com/kurabuchi-kentaro/git-snag/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
