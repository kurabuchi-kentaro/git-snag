package gitcli

import (
	"testing"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

func branchesByName(t *testing.T, repoPath string) map[string]domain.Branch {
	t.Helper()
	list, err := New().Branches(t.Context(), repoPath)
	if err != nil {
		t.Fatalf("Branches: %v", err)
	}
	m := map[string]domain.Branch{}
	for _, b := range list {
		m[b.Name] = b
	}
	return m
}

func TestBranches_listsLocalBranchesWithStatus(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	fx.SetupOrigin()
	wt := fx.AddWorktree("feature/wt")
	fx.Git("branch", "bare")
	ahead := fx.AddWorktree("feature/ahead")
	fx.GitIn(ahead, "push", "-q", "-u", "origin", "feature/ahead")
	fx.CommitIn(ahead, "unpushed work")

	got := branchesByName(t, fx.Dir)
	if len(got) != 4 {
		t.Fatalf("branches = %v, want 4 entries", got)
	}
	if got["main"].WorktreePath != fx.Dir {
		t.Errorf("main.WorktreePath = %q, want the main worktree %q", got["main"].WorktreePath, fx.Dir)
	}
	if got["feature/wt"].WorktreePath != wt {
		t.Errorf("feature/wt.WorktreePath = %q, want %q", got["feature/wt"].WorktreePath, wt)
	}
	if b := got["bare"]; b.HasWorktree() {
		t.Errorf("bare should have no worktree: %+v", b)
	}
	if b := got["feature/ahead"]; b.UnpushedCount != 1 || b.UpstreamGone {
		t.Errorf("feature/ahead should be 1 ahead of a live upstream: %+v", b)
	}
	for name, b := range got {
		if b.LastCommitTime.IsZero() {
			t.Errorf("%s.LastCommitTime should be set", name)
		}
	}
}

func TestBranches_flagsGoneUpstream(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	fx.SetupOrigin()
	fx.Git("branch", "was-pushed")
	fx.Git("push", "-q", "-u", "origin", "was-pushed")
	fx.Git("push", "-q", "origin", "--delete", "was-pushed")

	got := branchesByName(t, fx.Dir)
	b := got["was-pushed"]
	if !b.UpstreamGone {
		t.Errorf("was-pushed should report a gone upstream: %+v", b)
	}
	if got["main"].UpstreamGone {
		t.Error("main upstream should not be gone")
	}
}

func TestBranches_repoWithoutOrigin(t *testing.T) {
	t.Parallel()
	fx := testutil.NewRepo(t)
	fx.Git("branch", "local-only")

	got := branchesByName(t, fx.Dir)
	if len(got) != 2 {
		t.Fatalf("branches = %v, want main and local-only", got)
	}
	if got["local-only"].UnpushedCount != 0 || got["local-only"].UpstreamGone {
		t.Errorf("local-only should have no upstream state: %+v", got["local-only"])
	}
}
