package gitcli

import (
	"errors"
	"testing"

	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

func TestDefaultBranch_originHeadConfigured(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	repo.SetupOrigin()

	got, err := New().DefaultBranch(t.Context(), repo.Dir)
	if err != nil {
		t.Fatalf("DefaultBranch: %v", err)
	}
	if got != "main" {
		t.Errorf("DefaultBranch = %q, want %q", got, "main")
	}
}

func TestDefaultBranch_noOriginReturnsErrNoDefaultBranch(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)

	_, err := New().DefaultBranch(t.Context(), repo.Dir)
	if !errors.Is(err, ErrNoDefaultBranch) {
		t.Fatalf("err = %v, want ErrNoDefaultBranch", err)
	}
}

func TestUnpushedCount_branchAheadOfUpstream(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	repo.SetupOrigin()
	wt := repo.AddWorktree("feature/ahead")
	repo.GitIn(wt, "push", "-q", "-u", "origin", "feature/ahead")
	repo.CommitIn(wt, "local one")
	repo.CommitIn(wt, "local two")

	count, hasUpstream, err := New().UnpushedCount(t.Context(), wt)
	if err != nil {
		t.Fatalf("UnpushedCount: %v", err)
	}
	if !hasUpstream {
		t.Error("branch with upstream should report HasUpstream=true")
	}
	if count != 2 {
		t.Errorf("UnpushedCount = %d, want 2", count)
	}
}

func TestUnpushedCount_noUpstreamIsNotAnError(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/local-only")

	count, hasUpstream, err := New().UnpushedCount(t.Context(), wt)
	if err != nil {
		t.Fatalf("UnpushedCount without upstream should not error: %v", err)
	}
	if hasUpstream {
		t.Error("branch without upstream should report HasUpstream=false")
	}
	if count != 0 {
		t.Errorf("UnpushedCount = %d, want 0", count)
	}
}

func TestUnpushedCount_detachedHeadIsNotAnError(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddDetachedWorktree()

	count, hasUpstream, err := New().UnpushedCount(t.Context(), wt)
	if err != nil {
		t.Fatalf("UnpushedCount on detached HEAD should not error: %v", err)
	}
	if hasUpstream || count != 0 {
		t.Errorf("detached HEAD: count=%d hasUpstream=%v, want 0/false", count, hasUpstream)
	}
}
