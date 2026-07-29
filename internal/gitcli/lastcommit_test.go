package gitcli

import (
	"testing"
	"time"

	"github.com/kurabuchi-kentaro/git-snag/internal/testutil"
)

func TestLastCommitTime_returnsHeadCommitTime(t *testing.T) {
	t.Parallel()
	repo := testutil.NewRepo(t)
	wt := repo.AddWorktree("feature/timed")
	before := time.Now().Add(-5 * time.Second)
	repo.CommitIn(wt, "the commit under test")
	after := time.Now().Add(5 * time.Second)

	got, err := New().LastCommitTime(t.Context(), wt)
	if err != nil {
		t.Fatalf("LastCommitTime: %v", err)
	}
	if got.Before(before) || got.After(after) {
		t.Errorf("LastCommitTime = %v, want between %v and %v", got, before, after)
	}
}
