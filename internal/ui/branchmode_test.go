package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/kurabuchi-kentaro/git-snag/internal/action"
	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

func goneBranch() func(*domain.Branch) {
	return func(b *domain.Branch) { b.UpstreamGone = true }
}

func mergedBranch() func(*domain.Branch) {
	return func(b *domain.Branch) { b.Merged = true }
}

// branchLeafRows collects the branch names of every leaf row.
func branchLeafRows(m Model) []string {
	var names []string
	for _, r := range m.rows {
		if r.kind == rowLeaf && r.node.Branch != nil {
			names = append(names, r.node.Branch.Name)
		}
	}
	return names
}

func TestBranchMode_toggleShowsBareBranchesAndBadge(t *testing.T) {
	t.Parallel()
	repo := withBareBranch(testRepo("alpha", "feature/x"), "feature/bare")
	m := modelWith(t, repo)

	view := stripANSI(m.View().Content)
	if strings.Contains(view, "feature/bare") {
		t.Errorf("worktree mode must not list bare branches:\n%s", view)
	}

	m = apply(t, m, press('b'))
	if !m.branchMode {
		t.Fatal("b should enter branch mode")
	}
	view = stripANSI(m.View().Content)
	if !strings.Contains(view, "bare") {
		t.Errorf("branch mode should list the bare branch:\n%s", view)
	}
	if !strings.Contains(view, "BRANCHES") {
		t.Errorf("footer should carry the BRANCHES badge:\n%s", view)
	}
	if !strings.Contains(view, "3 branches") {
		t.Errorf("counts should use the branches unit:\n%s", view)
	}

	m = apply(t, m, press('b'))
	if m.branchMode {
		t.Error("second b should return to worktree mode")
	}
}

func TestBranchMode_defaultBranchAnchorsTheTree(t *testing.T) {
	t.Parallel()
	m := apply(t, modelWith(t, testRepo("alpha", "feature/x")), press('b'))
	// rows: repo header, main anchor, feature/ group, x leaf
	if len(m.rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(m.rows))
	}
	anchor := m.rows[1]
	if anchor.kind != rowLeaf || anchor.node.Branch == nil || anchor.node.Branch.Name != "main" {
		t.Fatalf("row 1 = %+v, want the main anchor", anchor.node)
	}
	if len(anchor.node.Children) == 0 {
		t.Error("the anchor should carry the subtree")
	}
	if m.rows[2].kind != rowGroup {
		t.Errorf("row 2 should be the feature/ group, got %+v", m.rows[2].node)
	}
}

func TestBranchMode_worktreeBranchCarriesPlusMark(t *testing.T) {
	t.Parallel()
	repo := withBareBranch(testRepo("alpha", "feature/wt"), "feature/bare")
	m := apply(t, modelWith(t, repo), press('b'))
	view := stripANSI(m.View().Content)
	if !strings.Contains(view, "+ wt") {
		t.Errorf("worktree branch should carry the + mark:\n%s", view)
	}
	if strings.Contains(view, "+ bare") {
		t.Errorf("bare branch must not carry the + mark:\n%s", view)
	}
}

func TestBranchMode_selectionAndPlan(t *testing.T) {
	t.Parallel()
	repo := withBareBranch(testRepo("alpha", "feature/wt"), "feature/bare")
	m := apply(t, modelWith(t, repo), press('b'))

	// Select both deletable branches via the repo header (anchor is
	// protected and must be skipped).
	m.focus = 0
	m = apply(t, m, space())
	if len(m.selection) != 2 {
		t.Fatalf("selection = %v, want the two deletable branches", m.selection)
	}
	if m.selection[branchKey("/work/alpha", "main")] {
		t.Error("the protected default branch must not be selected")
	}

	plan := m.buildPlan()
	if len(plan) != 2 {
		t.Fatalf("plan = %+v, want 2 items", plan)
	}
	for _, it := range plan {
		switch {
		case it.BranchOnly():
			if it.Branch != "feature/bare" {
				t.Errorf("branch-only item = %+v, want feature/bare", it)
			}
		default:
			if it.Worktree.Branch != "feature/wt" || !it.DeleteBranch {
				t.Errorf("worktree item = %+v, want feature/wt with branch deletion", it)
			}
		}
	}
}

func TestBranchMode_switchClearsSelectionBothWays(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	focusOn(t, &m, leafNamed("x"))
	m = apply(t, m, space())
	if len(m.selection) == 0 {
		t.Fatal("setup: worktree selected")
	}
	m = apply(t, m, press('b'))
	if len(m.selection) != 0 {
		t.Errorf("entering branch mode should clear the selection: %v", m.selection)
	}
	m.focus = 0
	m = apply(t, m, space())
	if len(m.selection) == 0 {
		t.Fatal("setup: branches selected")
	}
	m = apply(t, m, press('b'))
	if len(m.selection) != 0 {
		t.Errorf("leaving branch mode should clear the selection: %v", m.selection)
	}
}

func TestBranchMode_flatSortListsCandidatesAcrossRepos(t *testing.T) {
	t.Parallel()
	oldRepo := withBareBranch(testRepo("alpha"), "feature/old", func(b *domain.Branch) {
		b.LastCommitTime = time.Unix(1_500_000_000, 0)
	})
	freshRepo := withBareBranch(testRepo("beta"), "feature/new", func(b *domain.Branch) {
		b.LastCommitTime = time.Unix(1_650_000_000, 0)
	})
	m := apply(t, modelWith(t, oldRepo, freshRepo), press('b'), press('s'))

	names := branchLeafRows(m)
	if len(names) != 2 || names[0] != "feature/old" || names[1] != "feature/new" {
		t.Errorf("flat order = %v, want oldest first across repos", names)
	}
	for _, r := range m.rows {
		if r.kind == rowRepo {
			t.Error("global flat branch list should have no repo headers")
		}
	}
	if view := stripANSI(m.View().Content); strings.Contains(view, "main") && strings.Contains(view, "+ main") {
		t.Errorf("protected branches must be hidden in flat mode:\n%s", view)
	}
}

func TestBranchMode_mergedFilterIncludesGoneUpstreams(t *testing.T) {
	t.Parallel()
	repo := withBareBranch(testRepo("alpha"), "feature/squashed", goneBranch())
	repo = withBareBranch(repo, "feature/merged", mergedBranch())
	repo = withBareBranch(repo, "feature/active")
	m := apply(t, modelWith(t, repo), press('b'), press('m'))

	names := branchLeafRows(m)
	for _, n := range names {
		if n == "feature/active" {
			t.Errorf("active branch should be filtered out: %v", names)
		}
	}
	found := map[string]bool{}
	for _, n := range names {
		found[n] = true
	}
	if !found["feature/squashed"] || !found["feature/merged"] {
		t.Errorf("merged-or-gone branches should remain: %v", names)
	}
}

func TestBranchMode_applyResultsRemovesDeletedBranches(t *testing.T) {
	t.Parallel()
	repo := withBareBranch(testRepo("alpha"), "feature/doomed")
	m := apply(t, modelWith(t, repo), press('b'))
	m.applyResults(DeleteResultsMsg{Results: []action.Result{{
		Item:            action.PlanItem{RepoPath: "/work/alpha", Branch: "feature/doomed"},
		BranchAttempted: true,
	}}})
	for _, b := range m.repos[0].Branches {
		if b.Name == "feature/doomed" {
			t.Error("a deleted branch must leave the model")
		}
	}
	if m.phase != phaseSummary {
		t.Errorf("phase = %v, want summary", m.phase)
	}
}

func TestBranchMode_worktreeDeletionUpdatesBranchList(t *testing.T) {
	t.Parallel()
	m := apply(t, modelWith(t, testRepo("alpha", "feature/wt")), press('b'))
	wt := m.repos[0].Worktrees[1]
	m.applyResults(DeleteResultsMsg{Results: []action.Result{{
		Item:            action.PlanItem{RepoPath: "/work/alpha", Worktree: wt, DeleteBranch: true},
		BranchAttempted: true,
	}}})
	for _, b := range m.repos[0].Branches {
		if b.Name == "feature/wt" {
			t.Error("the branch deleted alongside its worktree must leave the model")
		}
	}
}
