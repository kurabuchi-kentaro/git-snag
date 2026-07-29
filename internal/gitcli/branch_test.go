package gitcli

import (
	"strings"
	"testing"

	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

// mergedBranch creates a branch pointing at the current main HEAD (thus
// already merged) without checking it out anywhere.
func mergedBranch(repo *testutil.Repo, name string) {
	repo.Git("branch", name)
}

// unmergedBranch creates a branch with a commit main does not have, and
// leaves it checked out nowhere.
func unmergedBranch(repo *testutil.Repo, name string) {
	wt := repo.AddWorktree(name)
	repo.CommitIn(wt, "diverge "+name)
	repo.Git("worktree", "remove", wt)
}

func TestDeleteBranch_mergedBranchSucceeds(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	mergedBranch(repo, "feature/merged")

	if err := New().DeleteBranch(t.Context(), repo.Dir, "feature/merged", false); err != nil {
		t.Fatalf("DeleteBranch: %v", err)
	}
	if out := repo.Git("branch", "--list", "feature/merged"); strings.TrimSpace(out) != "" {
		t.Errorf("branch still exists:\n%s", out)
	}
}

func TestDeleteBranch_unmergedWithoutForceFails(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	unmergedBranch(repo, "feature/unmerged")

	err := New().DeleteBranch(t.Context(), repo.Dir, "feature/unmerged", false)
	if err == nil {
		t.Fatal("expected deleting an unmerged branch without force to fail")
	}
	if out := repo.Git("branch", "--list", "feature/unmerged"); strings.TrimSpace(out) == "" {
		t.Error("unmerged branch should still exist after failed delete")
	}
}

func TestDeleteBranch_unmergedWithForceSucceeds(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	unmergedBranch(repo, "feature/unmerged-force")

	if err := New().DeleteBranch(t.Context(), repo.Dir, "feature/unmerged-force", true); err != nil {
		t.Fatalf("DeleteBranch(force): %v", err)
	}
	if out := repo.Git("branch", "--list", "feature/unmerged-force"); strings.TrimSpace(out) != "" {
		t.Errorf("branch still exists:\n%s", out)
	}
}

func TestMergedBranches_containsMergedOmitsUnmerged(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	mergedBranch(repo, "feature/in")
	unmergedBranch(repo, "feature/out")

	got, err := New().MergedBranches(t.Context(), repo.Dir, "main")
	if err != nil {
		t.Fatalf("MergedBranches: %v", err)
	}
	if !got["feature/in"] {
		t.Errorf("feature/in should be reported merged: %v", got)
	}
	if got["feature/out"] {
		t.Errorf("feature/out should not be reported merged: %v", got)
	}
}
