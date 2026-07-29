package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kurabuchi-kentaro/git-snag/internal/action"
	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

func TestDelete_withoutSelectionDoesNotOpenConfirm(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	m = apply(t, m, press('d'))
	if m.phase != phaseBrowsing {
		t.Errorf("phase = %v, want browsing", m.phase)
	}
}

func TestDelete_opensConfirmListingWarnings(t *testing.T) {
	t.Parallel()
	repo := testRepo("alpha", "feature/dirty", "feature/pushy")
	repo.Worktrees[1].Dirty = true
	repo.Worktrees[1].Locked = true
	repo.Worktrees[2].UnpushedCount = 3
	repo.Worktrees[2].HasUpstream = true
	m := modelWith(t, repo)
	focusOn(t, &m, groupNamed("feature"))
	m = apply(t, m, space(), press('d'))

	if m.phase != phaseConfirming {
		t.Fatalf("phase = %v, want confirming", m.phase)
	}
	if len(m.confirmItems) != 2 {
		t.Fatalf("confirm items = %d, want 2", len(m.confirmItems))
	}
	view := m.View().Content
	for _, want := range []string{
		"Delete 2 worktree(s)?",
		"uncommitted changes",
		"locked (will force-unlock)",
		"3 commit(s) not pushed",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("confirm view missing %q:\n%s", want, view)
		}
	}
}

func TestConfirm_spaceTogglesBranchExclusion(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))
	focusOn(t, &m, groupNamed("feature"))
	m = apply(t, m, space(), press('d'))

	m = apply(t, m, press('j'), space())
	if m.confirmItems[0].DeleteBranch != true {
		t.Error("first item should keep branch deletion on")
	}
	if m.confirmItems[1].DeleteBranch != false {
		t.Error("second item should have branch deletion excluded")
	}
	m = apply(t, m, space())
	if m.confirmItems[1].DeleteBranch != true {
		t.Error("space should toggle exclusion back")
	}
}

func TestConfirm_cancelReturnsToBrowsingWithoutDeleting(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space(), press('d'), press('n'))
	if m.phase != phaseBrowsing {
		t.Errorf("phase = %v, want browsing after cancel", m.phase)
	}
	if len(m.selection) == 0 {
		t.Error("cancel should keep the selection")
	}
}

func TestConfirm_yesProducesExecuteCommand(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space(), press('d'))
	next, cmd := m.Update(press('y'))
	got := next.(Model)
	if got.phase != phaseExecuting {
		t.Errorf("phase = %v, want executing", got.phase)
	}
	if cmd == nil {
		t.Fatal("y should produce the delete command")
	}
}

func TestSummary_reportsPerItemAndSeparateBranchOutcome(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/ok", "feature/bad"))
	var okWt, badWt domain.Worktree
	for _, w := range m.repos[0].Worktrees {
		switch w.Branch {
		case "feature/ok":
			okWt = w
		case "feature/bad":
			badWt = w
		}
	}
	msg := DeleteResultsMsg{Results: []action.Result{
		{
			Item:            action.PlanItem{Worktree: okWt, DeleteBranch: true},
			BranchAttempted: true,
			BranchErr:       errors.New("branch not found"),
		},
		{
			Item:        action.PlanItem{Worktree: badWt, DeleteBranch: true},
			WorktreeErr: errors.New("permission denied"),
		},
	}}
	m = apply(t, m, msg)

	if m.phase != phaseSummary {
		t.Fatalf("phase = %v, want summary", m.phase)
	}
	view := m.View().Content
	for _, want := range []string{
		"✓ feature/ok: worktree removed",
		"✗ branch deletion failed",
		"✗ feature/bad: worktree removal failed",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("summary missing %q:\n%s", want, view)
		}
	}

	// The successfully removed worktree leaves the tree; the failed one stays.
	branches := map[string]bool{}
	for _, w := range m.repos[0].Worktrees {
		branches[w.Branch] = true
	}
	if branches["feature/ok"] {
		t.Error("removed worktree should leave the model")
	}
	if !branches["feature/bad"] {
		t.Error("failed worktree should remain in the model")
	}
	if len(m.selection) != 0 {
		t.Error("selection should be cleared after execution")
	}

	m = apply(t, m, press('x'))
	if m.phase != phaseBrowsing {
		t.Errorf("any key should leave the summary, phase = %v", m.phase)
	}
	_ = time.Now
}
