package gitcli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// DefaultBranch resolves the repository's default branch from origin/HEAD.
// When it cannot be resolved (typically: no origin remote), it returns an
// error matching ErrNoDefaultBranch so callers can skip merge detection
// (REQ: no-origin contract; see ADR 0009).
func (c *Client) DefaultBranch(ctx context.Context, repoPath string) (string, error) {
	out, err := c.run(ctx, repoPath, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", fmt.Errorf("resolving origin/HEAD in %s: %w: %w", repoPath, ErrNoDefaultBranch, err)
	}
	return strings.TrimPrefix(out, "refs/remotes/origin/"), nil
}

// UnpushedCount returns how many commits the worktree's HEAD is ahead of its
// upstream. A branch without an upstream, or a detached HEAD, is not an
// error: it reports (0, false, nil).
func (c *Client) UnpushedCount(ctx context.Context, wtPath string) (count int, hasUpstream bool, err error) {
	out, err := c.run(ctx, wtPath, "rev-list", "--count", "@{upstream}..HEAD")
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "no upstream configured") ||
			strings.Contains(msg, "does not point to a branch") ||
			strings.Contains(msg, "no such branch") {
			return 0, false, nil
		}
		return 0, false, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return 0, false, fmt.Errorf("parsing rev-list count %q: %w", out, err)
	}
	return n, true, nil
}
