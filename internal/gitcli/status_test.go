package gitcli

import (
	"testing"

	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

func TestIsDirty_cleanWorktreeIsFalse(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/clean")

	dirty, err := New().IsDirty(t.Context(), wt)
	if err != nil {
		t.Fatalf("IsDirty: %v", err)
	}
	if dirty {
		t.Error("clean worktree reported dirty")
	}
}

func TestIsDirty_modifiedTrackedFileIsTrue(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/tracked")
	repo.WriteFile(wt, "file.txt", "v1")
	repo.GitIn(wt, "add", "file.txt")
	repo.GitIn(wt, "commit", "-q", "-m", "add file")
	repo.WriteFile(wt, "file.txt", "v2")

	dirty, err := New().IsDirty(t.Context(), wt)
	if err != nil {
		t.Fatalf("IsDirty: %v", err)
	}
	if !dirty {
		t.Error("worktree with modified tracked file reported clean")
	}
}

func TestIsDirty_untrackedFileIsTrue(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/untracked")
	repo.WriteFile(wt, "new.txt", "hello")

	dirty, err := New().IsDirty(t.Context(), wt)
	if err != nil {
		t.Fatalf("IsDirty: %v", err)
	}
	if !dirty {
		t.Error("worktree with untracked file reported clean")
	}
}
