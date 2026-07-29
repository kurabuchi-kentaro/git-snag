// Command git-snag is a TUI for cleaning up git worktrees and their branches
// across repositories found under the current directory.
package main

import (
	"fmt"

	"github.com/kurabuchi-kentaro/git-snag/internal/version"
)

func main() {
	// Phase 0 placeholder: the real entrypoint (internal/cli.Run) arrives
	// with the first UI milestone.
	fmt.Printf("git-snag %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
}
