// Package action executes confirmed deletion batches. It is the only place
// that mutates repositories, and it is invoked from the UI exclusively as a
// tea.Cmd so the UI itself stays free of process spawning.
package action

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
	"github.com/kurabuchi-kentaro/git-snag/internal/gitcli"
)

// PlanItem is one confirmed deletion: a worktree, and whether its branch
// goes with it (the confirmation screen can exclude the branch per item).
// Alternatively, Branch names a branch-only deletion (branch mode, ADR
// 0014): no worktree is touched, and Worktree/DeleteBranch stay zero.
type PlanItem struct {
	RepoPath     string
	Worktree     domain.Worktree
	DeleteBranch bool
	Branch       string
}

// BranchOnly reports whether the item deletes a branch without a worktree.
func (it PlanItem) BranchOnly() bool { return it.Branch != "" }

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

// maxConcurrentGit caps the git processes a batch runs at once. Removal is
// dominated by unlinking files, and beyond a few concurrent removals the
// filesystem contention outweighs the overlap (measured on ext4 under WSL2:
// 4 was fastest, 12 slower than 2).
const maxConcurrentGit = 4

// Execute runs the batch, never aborting on individual failures: every item
// gets a Result, in item order. Repositories are processed concurrently, and
// within a repository worktree removals run concurrently too; the steps that
// rewrite repository-wide state run afterwards, one at a time per repository:
// prune (which would race a removal in flight) and branch deletions (which
// contend for the ref locks). Prunable entries are pruned once per repository
// (git offers no per-path prune), then verified against a fresh listing.
func Execute(ctx context.Context, items []PlanItem) []Result {
	git := gitcli.New()
	results := make([]Result, len(items))
	byRepo := map[string][]int{}
	var repoOrder []string
	for i, it := range items {
		results[i] = Result{Item: it}
		if _, seen := byRepo[it.RepoPath]; !seen {
			repoOrder = append(repoOrder, it.RepoPath)
		}
		byRepo[it.RepoPath] = append(byRepo[it.RepoPath], i)
	}

	slots := make(chan struct{}, maxConcurrentGit)
	var wg sync.WaitGroup
	for _, repo := range repoOrder {
		wg.Go(func() { executeRepo(ctx, git, slots, byRepo[repo], results) })
	}
	wg.Wait()
	return results
}

// executeRepo runs one repository's items, writing only results[idx] for
// its own indices. slots bounds the git processes running across the batch.
func executeRepo(ctx context.Context, git *gitcli.Client, slots chan struct{}, idx []int, results []Result) {
	acquire := func() bool {
		select {
		case slots <- struct{}{}:
			return true
		case <-ctx.Done():
			return false
		}
	}
	release := func() { <-slots }

	var wg sync.WaitGroup
	var prunable []int
	for _, i := range idx {
		r := &results[i]
		switch {
		case r.Item.BranchOnly():
		case r.Item.Worktree.Prunable:
			prunable = append(prunable, i)
		default:
			wg.Go(func() {
				if ctx.Err() != nil || !acquire() {
					r.Skipped = true
					return
				}
				defer release()
				r.WorktreeErr = removeWorktree(ctx, git, r.Item)
				if r.WorktreeErr == nil {
					removeEmptyParents(r.Item.Worktree.Path, r.Item.RepoPath)
				}
			})
		}
	}
	wg.Wait()

	if !acquire() {
		for _, i := range idx {
			if it := results[i].Item; it.BranchOnly() || it.Worktree.Prunable {
				results[i].Skipped = true
			}
		}
		return
	}
	defer release()

	pruned := map[string]error{}
	for _, i := range prunable {
		r := &results[i]
		if ctx.Err() != nil {
			r.Skipped = true
			continue
		}
		r.WorktreeErr = pruneOnce(ctx, git, pruned, r.Item)
		if r.WorktreeErr == nil {
			removeEmptyParents(r.Item.Worktree.Path, r.Item.RepoPath)
		}
	}

	for _, i := range idx {
		r := &results[i]
		it := r.Item
		if it.BranchOnly() {
			if ctx.Err() != nil {
				r.Skipped = true
				continue
			}
			r.BranchAttempted = true
			r.BranchForced, r.BranchErr = deleteBranch(ctx, git, it.RepoPath, it.Branch)
			continue
		}
		if r.Skipped || r.WorktreeErr != nil || !it.DeleteBranch || it.Worktree.Detached() {
			continue
		}
		r.BranchAttempted = true
		r.BranchForced, r.BranchErr = deleteBranch(ctx, git, it.RepoPath, it.Worktree.Branch)
	}
}

// removeWorktree unlocks when needed and removes with --force: the batch was
// explicitly confirmed past the dirty/locked warnings (see ADR 0005), so a
// plain-removal refusal is not a stop condition. Going straight to --force
// also skips the full status scan a plain removal runs first.
func removeWorktree(ctx context.Context, git *gitcli.Client, it PlanItem) error {
	if it.Worktree.Locked {
		if err := git.UnlockWorktree(ctx, it.RepoPath, it.Worktree.Path); err != nil {
			return err
		}
	}
	return git.RemoveWorktree(ctx, it.RepoPath, it.Worktree.Path, true)
}

// removeEmptyParents deletes the nesting directories a slash-named worktree
// leaves behind (feature/, feature/fix/, ...): it climbs from the removed
// path's parent, stopping at the first directory that is not empty, at the
// repository, or at the filesystem root. os.Remove refuses to delete
// non-empty directories, so the climb cannot take anything that still holds
// content — the error is the stop condition, not a failure worth reporting.
func removeEmptyParents(path, repoPath string) {
	for dir := filepath.Dir(path); dir != repoPath && filepath.Dir(dir) != dir; dir = filepath.Dir(dir) {
		if os.Remove(dir) != nil {
			return
		}
	}
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
func deleteBranch(ctx context.Context, git *gitcli.Client, repoPath, branch string) (forced bool, err error) {
	if err = git.DeleteBranch(ctx, repoPath, branch, false); err == nil {
		return false, nil
	}
	if forceErr := git.DeleteBranch(ctx, repoPath, branch, true); forceErr == nil {
		return true, nil
	}
	return false, err
}
