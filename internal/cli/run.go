// Package cli parses flags, validates the environment, and wires the scan
// pipeline into the Bubble Tea program.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/enrich"
	"github.com/kurabuchi-kentaro/git-snag/internal/scan"
	"github.com/kurabuchi-kentaro/git-snag/internal/ui"
	"github.com/kurabuchi-kentaro/git-snag/internal/version"
)

// Run is the process entrypoint behind cmd/git-snag. It returns the exit
// code instead of calling os.Exit so it stays testable.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("git-snag", flag.ContinueOnError)
	fs.SetOutput(stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: git-snag [flags] [scan-root]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "git-snag %s (commit %s, built %s)\n", version.Version, version.Commit, version.Date)
		return 0
	}

	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(stderr, "git-snag: git executable not found in PATH")
		return 1
	}

	root := "."
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}
	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(stderr, "git-snag: scan root %s: %v\n", root, err)
		return 1
	}
	if !info.IsDir() {
		fmt.Fprintf(stderr, "git-snag: scan root %s is not a directory\n", root)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "git-snag: %v\n", err)
		return 1
	}

	if err := runTUI(root, cwd); err != nil {
		fmt.Fprintf(stderr, "git-snag: %v\n", err)
		return 1
	}
	return 0
}

// runTUI starts the Bubble Tea program and feeds it the scan stream.
func runTUI(root, cwd string) error {
	p := tea.NewProgram(ui.NewModel())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer p.Send(ui.ScanDoneMsg{})
		ch, err := scan.Walk(ctx, root, scan.DefaultExcludes())
		if err != nil {
			return
		}
		enricher := enrich.New(cwd)
		for repoPath := range ch {
			repo, err := enricher.Enrich(ctx, repoPath)
			if err != nil {
				continue
			}
			p.Send(ui.RepoFoundMsg{Repo: repo})
		}
	}()

	_, err := p.Run()
	return err
}
