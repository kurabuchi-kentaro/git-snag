// Package gitcli wraps the local git binary via os/exec. It is the only
// package (besides testutil) that spawns git processes.
package gitcli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// gitExecutable is resolved through PATH at process start.
const gitExecutable = "git"

// ErrNotRepository marks git failures caused by running against a directory
// that is not (inside) a git repository. Callers branch on it via errors.Is.
var ErrNotRepository = errors.New("not a git repository")

// ErrNoDefaultBranch marks the absence of a resolvable default branch
// (typically: no origin remote). Merge detection is skipped when it occurs.
var ErrNoDefaultBranch = errors.New("no default branch")

// Client executes git commands. The zero value is not usable; call New.
type Client struct {
	git string
}

// New returns a Client that invokes the git binary found on PATH.
func New() *Client {
	return &Client{git: gitExecutable}
}

// run executes git in dir and returns trimmed stdout. Non-zero exits are
// wrapped with the invoked subcommand and git's stderr (REQ: error contract);
// "not a git repository" failures additionally match ErrNotRepository.
func (c *Client) run(ctx context.Context, dir string, args ...string) (string, error) {
	// #nosec G204 -- args are built from internal call sites; paths and
	// branch names are passed as argv entries, never through a shell.
	cmd := exec.CommandContext(ctx, c.git, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "not a git repository") {
			return "", fmt.Errorf("git %s in %s: %w: %s", args[0], dir, ErrNotRepository, msg)
		}
		return "", fmt.Errorf("git %s in %s: %w (stderr: %s)", strings.Join(args, " "), dir, err, msg)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}
