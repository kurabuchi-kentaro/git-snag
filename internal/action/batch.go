// Package action executes confirmed deletion batches. It is the only place
// that mutates repositories, and it is invoked from the UI exclusively as a
// tea.Cmd so the UI itself stays free of process spawning.
package action

import (
	"context"
	"errors"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
	"github.com/kurabuchi-kentaro/git-snag/internal/gitcli"
)

// PlanItem is one confirmed deletion: a worktree, and whether its branch
// goes with it (the confirmation screen can exclude the branch per item).
type PlanItem struct {
	RepoPath     string
	Worktree     domain.Worktree
	DeleteBranch bool
}

// Result records what happened to one item. Worktree removal and branch
// deletion are reported separately (REQ-A8): "worktree gone, branch delete
// failed" must be expressible in the summary.
type Result struct {
	Item PlanItem
	// Skipped is set when ctx was cancelled before the item ran.
	Skipped bool
	// WorktreeErr is nil when the worktree was removed (or pruned).
	WorktreeErr error
	// BranchAttempted reports whether a branch deletion was tried at all.
	BranchAttempted bool
	// BranchErr is nil when the branch deletion succeeded.
	BranchErr error
	// BranchForced marks the -D fallback: the branch had unmerged commits.
	BranchForced bool
}

// Execute runs the batch, never aborting on individual failures: every item
// gets a Result. Prunable entries are pruned once per repository (git offers
// no per-path prune), then verified against a fresh listing.
func Execute(ctx context.Context, items []PlanItem) []Result {
	git := gitcli.New()
	results := make([]Result, len(items))
	prunedRepos := map[string]error{}

	for i, it := range items {
		results[i] = Result{Item: it}
		if ctx.Err() != nil {
			results[i].Skipped = true
			continue
		}

		if it.Worktree.Prunable {
			results[i].WorktreeErr = pruneOnce(ctx, git, prunedRepos, it)
		} else {
			results[i].WorktreeErr = removeWorktree(ctx, git, it)
		}
		if results[i].WorktreeErr != nil {
			continue
		}

		if it.DeleteBranch && !it.Worktree.Detached() {
			results[i].BranchAttempted = true
			results[i].BranchForced, results[i].BranchErr = deleteBranch(ctx, git, it)
		}
	}
	return results
}

// removeWorktree unlocks when needed and removes, escalating to --force on
// failure: the batch was explicitly confirmed past the dirty/locked warnings
// (see ADR 0005), so a plain-removal refusal is not a stop condition.
func removeWorktree(ctx context.Context, git *gitcli.Client, it PlanItem) error {
	if it.Worktree.Locked {
		if err := git.UnlockWorktree(ctx, it.RepoPath, it.Worktree.Path); err != nil {
			return err
		}
	}
	err := git.RemoveWorktree(ctx, it.RepoPath, it.Worktree.Path, false)
	if err == nil {
		return nil
	}
	return git.RemoveWorktree(ctx, it.RepoPath, it.Worktree.Path, true)
}

// pruneOnce prunes a repository at most once per batch and attributes the
// outcome to every prunable item of that repository.
func pruneOnce(ctx context.Context, git *gitcli.Client, pruned map[string]error, it PlanItem) error {
	if err, done := pruned[it.RepoPath]; done {
		return err
	}
	err := git.PruneWorktrees(ctx, it.RepoPath)
	if err == nil {
		err = verifyPruned(ctx, git, it)
	}
	pruned[it.RepoPath] = err
	return err
}

// verifyPruned confirms the pruned entry really left the listing.
func verifyPruned(ctx context.Context, git *gitcli.Client, it PlanItem) error {
	wts, err := git.ListWorktrees(ctx, it.RepoPath)
	if err != nil {
		return err
	}
	for _, w := range wts {
		if w.Path == it.Worktree.Path {
			return errors.New("entry still present after prune")
		}
	}
	return nil
}

// deleteBranch tries a clean -d first and falls back to -D, reporting the
// escalation so the summary can distinguish clean from forced deletions.
func deleteBranch(ctx context.Context, git *gitcli.Client, it PlanItem) (forced bool, err error) {
	if err = git.DeleteBranch(ctx, it.RepoPath, it.Worktree.Branch, false); err == nil {
		return false, nil
	}
	if forceErr := git.DeleteBranch(ctx, it.RepoPath, it.Worktree.Branch, true); forceErr == nil {
		return true, nil
	}
	return false, err
}
