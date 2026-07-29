package gitcli

import (
	"context"
	"fmt"
	"strings"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// ListWorktrees returns every worktree of the repository at repoPath, parsed
// from `git worktree list --porcelain`. The first entry is the main worktree.
func (c *Client) ListWorktrees(ctx context.Context, repoPath string) ([]domain.Worktree, error) {
	out, err := c.run(ctx, repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktrees(out)
}

// parseWorktrees parses `git worktree list --porcelain` output. Entries are
// separated by blank lines. Unknown attribute lines are ignored for forward
// compatibility with newer git versions (REQ: parser contract); an entry
// without the mandatory `worktree` line yields an error identifying it.
func parseWorktrees(out string) ([]domain.Worktree, error) {
	var result []domain.Worktree
	entryIndex := 0
	for _, block := range strings.Split(out, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		entryIndex++
		var wt domain.Worktree
		sawWorktreeLine := false
		for _, line := range strings.Split(block, "\n") {
			key, value, _ := strings.Cut(line, " ")
			switch key {
			case "worktree":
				wt.Path = value
				sawWorktreeLine = true
			case "HEAD":
				wt.HeadSHA = value
			case "branch":
				wt.Branch = strings.TrimPrefix(value, "refs/heads/")
			case "detached":
				// Boolean attribute; Branch stays empty.
			case "locked":
				wt.Locked = true
			case "prunable":
				wt.Prunable = true
			case "bare":
				// Bare repositories are out of scope for v0.1 (ADR 0011);
				// tolerate the attribute so parsing does not fail.
			default:
				// Unknown attribute from a newer git: ignore.
			}
		}
		if !sawWorktreeLine {
			return nil, fmt.Errorf("worktree list entry %d has no worktree line", entryIndex)
		}
		wt.IsMain = len(result) == 0
		result = append(result, wt)
	}
	return result, nil
}

// RemoveWorktree removes the worktree at wtPath from the repository at
// repoPath. force is required by git when the worktree is dirty.
func (c *Client) RemoveWorktree(ctx context.Context, repoPath, wtPath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, wtPath)
	_, err := c.run(ctx, repoPath, args...)
	return err
}

// UnlockWorktree removes the lock on the worktree at wtPath.
func (c *Client) UnlockWorktree(ctx context.Context, repoPath, wtPath string) error {
	_, err := c.run(ctx, repoPath, "worktree", "unlock", wtPath)
	return err
}

// PruneWorktrees drops the administrative entries of all worktrees whose
// directories no longer exist. git offers no per-path pruning, so this acts
// on the whole repository (one call per repo per batch — REQ-P1).
func (c *Client) PruneWorktrees(ctx context.Context, repoPath string) error {
	_, err := c.run(ctx, repoPath, "worktree", "prune")
	return err
}
