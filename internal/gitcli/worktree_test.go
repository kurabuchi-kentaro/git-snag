package gitcli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

func TestListWorktrees_returnsMainAndLinkedWorktrees(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt1 := repo.AddWorktree("feature/one")
	wt2 := repo.AddWorktree("feature/two")

	got, err := New().ListWorktrees(t.Context(), repo.Dir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d worktrees, want 3: %+v", len(got), got)
	}
	if !got[0].IsMain {
		t.Error("first entry should be the main worktree")
	}
	if got[0].Branch != "main" {
		t.Errorf("main branch = %q, want %q", got[0].Branch, "main")
	}
	byBranch := map[string]string{}
	for _, w := range got {
		if w.HeadSHA == "" {
			t.Errorf("worktree %s has empty HEAD SHA", w.Path)
		}
		byBranch[w.Branch] = w.Path
	}
	if byBranch["feature/one"] != wt1 {
		t.Errorf("feature/one path = %q, want %q", byBranch["feature/one"], wt1)
	}
	if byBranch["feature/two"] != wt2 {
		t.Errorf("feature/two path = %q, want %q", byBranch["feature/two"], wt2)
	}
	for _, w := range got[1:] {
		if w.IsMain {
			t.Errorf("linked worktree %s should not be IsMain", w.Path)
		}
	}
}

func TestListWorktrees_lockedWorktreeIsFlagged(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/locked")
	repo.Git("worktree", "lock", "--reason", "keep me", wt)

	got, err := New().ListWorktrees(t.Context(), repo.Dir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	found := false
	for _, w := range got {
		if w.Path == wt {
			found = true
			if !w.Locked {
				t.Error("locked worktree should have Locked=true")
			}
		}
	}
	if !found {
		t.Fatalf("locked worktree %s not in list", wt)
	}
}

func TestListWorktrees_missingDirectoryIsPrunable(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/gone")
	if err := os.RemoveAll(wt); err != nil {
		t.Fatal(err)
	}

	got, err := New().ListWorktrees(t.Context(), repo.Dir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	found := false
	for _, w := range got {
		if w.Path == wt {
			found = true
			if !w.Prunable {
				t.Error("removed worktree should have Prunable=true")
			}
		}
	}
	if !found {
		t.Fatalf("prunable worktree %s not in list", wt)
	}
}

func TestListWorktrees_detachedWorktreeHasEmptyBranch(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddDetachedWorktree()

	got, err := New().ListWorktrees(t.Context(), repo.Dir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	for _, w := range got {
		if w.Path == wt {
			if w.Branch != "" {
				t.Errorf("detached worktree branch = %q, want empty", w.Branch)
			}
			if w.HeadSHA == "" {
				t.Error("detached worktree should still have a HEAD SHA")
			}
			return
		}
	}
	t.Fatalf("detached worktree %s not in list", wt)
}

func TestListWorktrees_nonRepositoryReturnsErrNotRepository(t *testing.T) {
	t.Parallel()
	_, err := New().ListWorktrees(t.Context(), t.TempDir())
	if !errors.Is(err, ErrNotRepository) {
		t.Fatalf("err = %v, want ErrNotRepository", err)
	}
}

func TestListWorktrees_corruptRepositoryReturnsStderr(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	// Corrupt the config so git fails with something other than
	// "not a repository" (a broken HEAD would make git treat the directory
	// as not-a-repo, which is the other error class).
	if err := os.WriteFile(filepath.Join(repo.Dir, ".git", "config"), []byte("[core\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := New().ListWorktrees(t.Context(), repo.Dir)
	if err == nil {
		t.Fatal("expected an error for a corrupt repository")
	}
	if errors.Is(err, ErrNotRepository) {
		t.Fatalf("corrupt repo should not map to ErrNotRepository: %v", err)
	}
}

func TestParseWorktrees_ignoresUnknownAttributeLines(t *testing.T) {
	t.Parallel()
	out := "worktree /repo\n" +
		"HEAD 1111111111111111111111111111111111111111\n" +
		"branch refs/heads/main\n" +
		"frobnicate future-attribute\n" +
		"\n"
	got, err := parseWorktrees(out)
	if err != nil {
		t.Fatalf("parseWorktrees: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d entries, want 1", len(got))
	}
	if got[0].Branch != "main" || got[0].Path != "/repo" {
		t.Errorf("known fields not parsed: %+v", got[0])
	}
}

func TestParseWorktrees_toleratesTrailingBlankLines(t *testing.T) {
	t.Parallel()
	out := "worktree /repo\n" +
		"HEAD 1111111111111111111111111111111111111111\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree /wt\n" +
		"HEAD 2222222222222222222222222222222222222222\n" +
		"branch refs/heads/feature/x\n" +
		"\n\n\n"
	got, err := parseWorktrees(out)
	if err != nil {
		t.Fatalf("parseWorktrees: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
}

func TestParseWorktrees_missingWorktreeLineReturnsEntryError(t *testing.T) {
	t.Parallel()
	out := "worktree /repo\n" +
		"HEAD 1111111111111111111111111111111111111111\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"HEAD 2222222222222222222222222222222222222222\n" +
		"branch refs/heads/broken\n" +
		"\n"
	_, err := parseWorktrees(out)
	if err == nil {
		t.Fatal("expected an error for an entry without a worktree line")
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("error should identify the broken entry (index 2): %v", err)
	}
}

func TestRemoveWorktree_cleanWorktreeIsRemoved(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/clean")

	if err := New().RemoveWorktree(t.Context(), repo.Dir, wt, false); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree directory still exists: %v", err)
	}
	if list := repo.Git("worktree", "list"); strings.Contains(list, wt) {
		t.Errorf("worktree still listed:\n%s", list)
	}
}

func TestRemoveWorktree_dirtyWithoutForceFails(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/dirty")
	repo.WriteFile(wt, "untracked.txt", "dirt")

	err := New().RemoveWorktree(t.Context(), repo.Dir, wt, false)
	if err == nil {
		t.Fatal("expected removal of a dirty worktree without force to fail")
	}
	if _, statErr := os.Stat(wt); statErr != nil {
		t.Errorf("dirty worktree should still exist: %v", statErr)
	}
}

func TestRemoveWorktree_dirtyWithForceSucceeds(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/dirty-force")
	repo.WriteFile(wt, "untracked.txt", "dirt")

	if err := New().RemoveWorktree(t.Context(), repo.Dir, wt, true); err != nil {
		t.Fatalf("RemoveWorktree(force): %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree directory still exists: %v", err)
	}
}

func TestUnlockThenRemove_lockedWorktreeSucceeds(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/locked-remove")
	repo.Git("worktree", "lock", wt)

	c := New()
	if err := c.UnlockWorktree(t.Context(), repo.Dir, wt); err != nil {
		t.Fatalf("UnlockWorktree: %v", err)
	}
	if err := c.RemoveWorktree(t.Context(), repo.Dir, wt, false); err != nil {
		t.Fatalf("RemoveWorktree after unlock: %v", err)
	}
	if _, err := os.Stat(wt); !os.IsNotExist(err) {
		t.Errorf("worktree directory still exists: %v", err)
	}
}

func TestPruneWorktrees_removesStaleEntryKeepsLive(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	gone := repo.AddWorktree("feature/gone")
	live := repo.AddWorktree("feature/live")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	if err := New().PruneWorktrees(t.Context(), repo.Dir); err != nil {
		t.Fatalf("PruneWorktrees: %v", err)
	}
	list := repo.Git("worktree", "list")
	if strings.Contains(list, gone) {
		t.Errorf("pruned worktree still listed:\n%s", list)
	}
	if !strings.Contains(list, live) {
		t.Errorf("live worktree disappeared:\n%s", list)
	}
	if _, err := os.Stat(live); err != nil {
		t.Errorf("live worktree directory damaged: %v", err)
	}
}

func TestRemoveWorktree_nonexistentPathFails(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)

	err := New().RemoveWorktree(t.Context(), repo.Dir, filepath.Join(t.TempDir(), "nope"), false)
	if err == nil {
		t.Fatal("expected removing a nonexistent worktree to fail")
	}
}
