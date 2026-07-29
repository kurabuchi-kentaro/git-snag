package enrich

import (
	"os"
	"testing"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

func enriched(t *testing.T, repoPath, cwd string) domain.Repo {
	t.Helper()
	repo, err := New(cwd).Enrich(t.Context(), repoPath)
	if err != nil {
		t.Fatalf("Enrich: %v", err)
	}
	return repo
}

func byBranch(t *testing.T, repo domain.Repo) map[string]domain.Worktree {
	t.Helper()
	m := map[string]domain.Worktree{}
	for _, wt := range repo.Worktrees {
		m[wt.Branch] = wt
	}
	return m
}

func TestEnrich_dirtyFlagsMatchWorktreeState(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	dirty := fx.AddWorktree("feature/dirty")
	fx.AddWorktree("feature/clean")
	fx.WriteFile(dirty, "mess.txt", "uncommitted")

	got := byBranch(t, enriched(t, fx.Dir, t.TempDir()))
	if !got["feature/dirty"].Dirty {
		t.Error("feature/dirty should be Dirty")
	}
	if got["feature/clean"].Dirty {
		t.Error("feature/clean should not be Dirty")
	}
}

func TestEnrich_mergedFlagsAgainstDefaultBranch(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	fx.SetupOrigin()
	fx.AddWorktree("feature/merged") // points at main HEAD: merged
	diverged := fx.AddWorktree("feature/diverged")
	fx.CommitIn(diverged, "diverge")

	got := byBranch(t, enriched(t, fx.Dir, t.TempDir()))
	if !got["feature/merged"].Merged {
		t.Error("feature/merged should be Merged")
	}
	if got["feature/diverged"].Merged {
		t.Error("feature/diverged should not be Merged")
	}
	if repo := enriched(t, fx.Dir, t.TempDir()); repo.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %q, want main", repo.DefaultBranch)
	}
}

func TestEnrich_unpushedCountsOnlyWithUpstream(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	fx.SetupOrigin()
	ahead := fx.AddWorktree("feature/ahead")
	fx.GitIn(ahead, "push", "-q", "-u", "origin", "feature/ahead")
	fx.CommitIn(ahead, "unpushed work")
	fx.AddWorktree("feature/no-upstream")

	got := byBranch(t, enriched(t, fx.Dir, t.TempDir()))
	if got["feature/ahead"].UnpushedCount != 1 || !got["feature/ahead"].HasUpstream {
		t.Errorf("feature/ahead = %+v, want UnpushedCount=1 HasUpstream=true", got["feature/ahead"])
	}
	if got["feature/no-upstream"].UnpushedCount != 0 || got["feature/no-upstream"].HasUpstream {
		t.Errorf("feature/no-upstream = %+v, want UnpushedCount=0 HasUpstream=false", got["feature/no-upstream"])
	}
}

func TestEnrich_currentWorktreeDetectedFromCwdSubdirectory(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	wt := fx.AddWorktree("feature/here")
	sub := wt + "/deep/inside"
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got := byBranch(t, enriched(t, fx.Dir, sub))
	if !got["feature/here"].IsCurrent {
		t.Error("worktree containing cwd should be IsCurrent")
	}
	if got["main"].IsCurrent {
		t.Error("main worktree should not be IsCurrent when cwd is elsewhere")
	}
}

func TestEnrich_mainWorktreeIsProtected(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	fx.AddWorktree("feature/other")

	got := byBranch(t, enriched(t, fx.Dir, t.TempDir()))
	main := got["main"]
	if !main.IsMain || !main.Protected() {
		t.Errorf("main worktree = %+v, want IsMain and Protected", main)
	}
	if got["feature/other"].Protected() {
		t.Error("plain linked worktree should not be Protected")
	}
}

func TestEnrich_noOriginSkipsMergeDetection(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	fx.AddWorktree("feature/local") // would be "merged" if detection ran

	repo := enriched(t, fx.Dir, t.TempDir())
	if repo.DefaultBranch != "" {
		t.Errorf("DefaultBranch = %q, want empty without origin", repo.DefaultBranch)
	}
	for _, wt := range repo.Worktrees {
		if wt.Merged {
			t.Errorf("worktree %s should not be Merged without a default branch", wt.Path)
		}
	}
}

func TestEnrich_manyWorktreesAllTaggedCorrectly(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	dirtyPaths := map[string]bool{}
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		wt := fx.AddWorktree("feature/" + name)
		if name == "c" || name == "f" {
			fx.WriteFile(wt, "dirt.txt", name)
			dirtyPaths[wt] = true
		}
	}

	repo := enriched(t, fx.Dir, t.TempDir())
	if len(repo.Worktrees) != 9 {
		t.Fatalf("got %d worktrees, want 9", len(repo.Worktrees))
	}
	for _, wt := range repo.Worktrees {
		if wt.IsMain {
			continue
		}
		if wt.Dirty != dirtyPaths[wt.Path] {
			t.Errorf("worktree %s Dirty=%v, want %v", wt.Path, wt.Dirty, dirtyPaths[wt.Path])
		}
		if wt.LastCommitTime.IsZero() {
			t.Errorf("worktree %s has zero LastCommitTime", wt.Path)
		}
	}
}

func TestEnrich_worktreeRemovedExternallyDoesNotAbortOthers(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	gone := fx.AddWorktree("feature/gone")
	fx.AddWorktree("feature/alive")
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}

	repo := enriched(t, fx.Dir, t.TempDir())
	got := byBranch(t, repo)
	if !got["feature/gone"].Prunable {
		t.Error("externally removed worktree should be Prunable")
	}
	alive := got["feature/alive"]
	if alive.LastCommitTime.IsZero() || alive.Prunable {
		t.Errorf("feature/alive not enriched correctly: %+v", alive)
	}
}
