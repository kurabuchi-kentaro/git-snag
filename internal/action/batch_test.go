package action

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
	"github.com/kurabuchi-kentaro/git-snag/internal/gitcli"
	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

func listWorktrees(t *testing.T, repoPath string) []domain.Worktree {
	t.Helper()
	wts, err := gitcli.New().ListWorktrees(t.Context(), repoPath)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	return wts
}

func findWt(t *testing.T, repoPath, branch string) domain.Worktree {
	t.Helper()
	for _, w := range listWorktrees(t, repoPath) {
		if w.Branch == branch {
			return w
		}
	}
	t.Fatalf("worktree for %s not found", branch)
	return domain.Worktree{}
}

func branchExists(repo *testutil.Repo, name string) bool {
	return strings.TrimSpace(repo.Git("branch", "--list", name)) != ""
}

func item(repoPath string, w domain.Worktree, deleteBranch bool) PlanItem {
	return PlanItem{RepoPath: repoPath, Worktree: w, DeleteBranch: deleteBranch}
}

func TestExecute_removesWorktreeAndBranch(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	repo.AddWorktree("feature/gone")
	w := findWt(t, repo.Dir, "feature/gone")

	results := Execute(t.Context(), []PlanItem{item(repo.Dir, w, true)})
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.WorktreeErr != nil || r.BranchErr != nil {
		t.Fatalf("result = %+v, want both steps successful", r)
	}
	if _, err := os.Stat(w.Path); !os.IsNotExist(err) {
		t.Error("worktree directory should be gone")
	}
	if branchExists(repo, "feature/gone") {
		t.Error("branch should be deleted")
	}
}

func TestExecute_branchExclusionKeepsBranch(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	repo.AddWorktree("feature/keep-branch")
	w := findWt(t, repo.Dir, "feature/keep-branch")

	results := Execute(t.Context(), []PlanItem{item(repo.Dir, w, false)})
	if r := results[0]; r.WorktreeErr != nil || r.BranchAttempted {
		t.Fatalf("result = %+v, want worktree removed and branch untouched", r)
	}
	if !branchExists(repo, "feature/keep-branch") {
		t.Error("branch should survive")
	}
}

func TestExecute_dirtyWorktreeIsForceRemoved(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/dirty")
	repo.WriteFile(wt, "mess.txt", "dirt")
	w := findWt(t, repo.Dir, "feature/dirty")
	w.Dirty = true

	results := Execute(t.Context(), []PlanItem{item(repo.Dir, w, true)})
	if r := results[0]; r.WorktreeErr != nil {
		t.Fatalf("dirty worktree should force-remove: %+v", r)
	}
}

func TestExecute_lockedWorktreeIsUnlockedThenRemoved(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/locked")
	repo.Git("worktree", "lock", wt)
	w := findWt(t, repo.Dir, "feature/locked")

	results := Execute(t.Context(), []PlanItem{item(repo.Dir, w, true)})
	if r := results[0]; r.WorktreeErr != nil {
		t.Fatalf("locked worktree should unlock+remove: %+v", r)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Error("worktree directory should be gone")
	}
}

func TestExecute_prunableEntriesPruneOncePerRepo(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	gone1 := repo.AddWorktree("feature/gone1")
	gone2 := repo.AddWorktree("feature/gone2")
	repo.AddWorktree("feature/live")
	for _, p := range []string{gone1, gone2} {
		if err := os.RemoveAll(p); err != nil {
			t.Fatal(err)
		}
	}
	w1 := findWt(t, repo.Dir, "feature/gone1")
	w2 := findWt(t, repo.Dir, "feature/gone2")
	live := findWt(t, repo.Dir, "feature/live")

	results := Execute(t.Context(), []PlanItem{
		item(repo.Dir, w1, true), item(repo.Dir, w2, true), item(repo.Dir, live, true),
	})
	for _, r := range results {
		if r.WorktreeErr != nil {
			t.Errorf("item %s failed: %v", r.Item.Worktree.Branch, r.WorktreeErr)
		}
	}
	remaining := listWorktrees(t, repo.Dir)
	if len(remaining) != 1 || !remaining[0].IsMain {
		t.Errorf("remaining worktrees = %+v, want only main", remaining)
	}
}

func TestExecute_mergedAndUnmergedBranchDeletionDistinguished(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	repo.AddWorktree("feature/merged")
	unmergedPath := repo.AddWorktree("feature/unmerged")
	repo.CommitIn(unmergedPath, "diverge")
	merged := findWt(t, repo.Dir, "feature/merged")
	unmerged := findWt(t, repo.Dir, "feature/unmerged")

	results := Execute(t.Context(), []PlanItem{
		item(repo.Dir, merged, true), item(repo.Dir, unmerged, true),
	})
	byBranch := map[string]Result{}
	for _, r := range results {
		byBranch[r.Item.Worktree.Branch] = r
	}
	if r := byBranch["feature/merged"]; r.BranchErr != nil || r.BranchForced {
		t.Errorf("merged branch should delete cleanly: %+v", r)
	}
	if r := byBranch["feature/unmerged"]; r.BranchErr != nil || !r.BranchForced {
		t.Errorf("unmerged branch should record the -D fallback: %+v", r)
	}
}

func TestExecute_detachedWorktreeSkipsBranchStep(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	repo.AddDetachedWorktree()
	var detached domain.Worktree
	for _, w := range listWorktrees(t, repo.Dir) {
		if w.Detached() && !w.IsMain {
			detached = w
		}
	}

	results := Execute(t.Context(), []PlanItem{item(repo.Dir, detached, true)})
	if r := results[0]; r.WorktreeErr != nil || r.BranchAttempted {
		t.Fatalf("detached worktree: %+v, want removal without branch step", r)
	}
}

func TestExecute_oneFailureDoesNotAbortTheBatch(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	repo := testutil.NewRepo(t)
	blockedParent := t.TempDir()
	blocked := blockedParent + "/blocked"
	repo.Git("worktree", "add", "-q", "-b", "feature/blocked", blocked)
	repo.AddWorktree("feature/fine")
	blockedWt := findWt(t, repo.Dir, "feature/blocked")
	fineWt := findWt(t, repo.Dir, "feature/fine")
	if err := os.Chmod(blockedParent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blockedParent, 0o755) })

	results := Execute(t.Context(), []PlanItem{
		item(repo.Dir, blockedWt, false), item(repo.Dir, fineWt, false),
	})
	byBranch := map[string]Result{}
	for _, r := range results {
		byBranch[r.Item.Worktree.Branch] = r
	}
	if byBranch["feature/blocked"].WorktreeErr == nil {
		t.Error("blocked item should fail")
	}
	if byBranch["feature/fine"].WorktreeErr != nil {
		t.Errorf("healthy item should still succeed: %+v", byBranch["feature/fine"])
	}
}

func TestExecute_worktreeGoneButBranchAlreadyDeleted(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	repo.AddWorktree("feature/ghost-branch")
	w := findWt(t, repo.Dir, "feature/ghost-branch")
	// Simulate the branch disappearing between listing and execution: remove
	// the worktree cleanly, then delete the branch, then re-add the path so
	// worktree removal succeeds but branch deletion cannot.
	repo.Git("worktree", "remove", w.Path)
	repo.Git("branch", "-D", "feature/ghost-branch")
	repo.Git("worktree", "add", "-q", "--detach", w.Path)
	w2 := findWt(t, repo.Dir, "")
	w2.Branch = "feature/ghost-branch" // stale metadata from the earlier listing

	results := Execute(t.Context(), []PlanItem{item(repo.Dir, w2, true)})
	r := results[0]
	if r.WorktreeErr != nil {
		t.Fatalf("worktree removal should succeed: %+v", r)
	}
	if r.BranchErr == nil || !r.BranchAttempted {
		t.Errorf("branch step should be recorded as failed separately: %+v", r)
	}
}

func TestExecute_emptyBatchIsNoOp(t *testing.T) {
	t.Parallel()
	results := Execute(t.Context(), nil)
	if len(results) != 0 {
		t.Errorf("results = %+v, want empty", results)
	}
}

func TestExecute_contextCancellationStopsRemainingItems(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	repo.AddWorktree("feature/never")
	w := findWt(t, repo.Dir, "feature/never")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	results := Execute(ctx, []PlanItem{item(repo.Dir, w, true)})
	if r := results[0]; !r.Skipped {
		t.Fatalf("result = %+v, want Skipped after cancellation", r)
	}
	if _, err := os.Stat(w.Path); err != nil {
		t.Error("worktree must be untouched after cancellation")
	}
}
