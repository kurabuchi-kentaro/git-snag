package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

func press(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: string(code)}
}

func apply(t *testing.T, m Model, msgs ...tea.Msg) Model {
	t.Helper()
	var model tea.Model = m
	for _, msg := range msgs {
		model, _ = model.Update(msg)
	}
	got, ok := model.(Model)
	if !ok {
		t.Fatalf("model type changed: %T", model)
	}
	return got
}

func testRepo(name string, branches ...string) domain.Repo {
	repo := domain.Repo{
		Path: "/work/" + name,
		Name: name,
	}
	repo.Worktrees = append(repo.Worktrees, domain.Worktree{
		Path: "/work/" + name, Branch: "main", IsMain: true,
		HeadSHA:        "abcdef0123456789abcdef0123456789abcdef01",
		LastCommitTime: time.Unix(1_700_000_000, 0),
	})
	for _, b := range branches {
		repo.Worktrees = append(repo.Worktrees, domain.Worktree{
			Path: "/work/" + name + "-wt/" + b, Branch: b,
			HeadSHA:        "abcdef0123456789abcdef0123456789abcdef01",
			LastCommitTime: time.Unix(1_600_000_000, 0),
		})
	}
	return repo
}

func modelWith(t *testing.T, repos ...domain.Repo) Model {
	t.Helper()
	m := NewModel()
	m.now = func() time.Time { return time.Unix(1_700_000_100, 0) }
	msgs := make([]tea.Msg, 0, len(repos)+1)
	for _, r := range repos {
		msgs = append(msgs, RepoFoundMsg{Repo: r})
	}
	msgs = append(msgs, ScanDoneMsg{})
	return apply(t, m, msgs...)
}

// --- streaming / empty state ---

func TestUpdate_repoFoundAppendsRows(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	if len(m.rows) == 0 {
		t.Fatal("rows should be built after RepoFoundMsg")
	}
	m = apply(t, m, RepoFoundMsg{Repo: testRepo("beta", "feature/y")})
	repoRows := 0
	for _, r := range m.rows {
		if r.kind == rowRepo {
			repoRows++
		}
	}
	if repoRows != 2 {
		t.Errorf("got %d repo headers, want 2", repoRows)
	}
}

func TestView_emptyScanShowsEmptyState(t *testing.T) {
	t.Parallel()
	m := modelWith(t) // no repos, scan done
	view := m.View().Content
	if !strings.Contains(view, "No worktrees") {
		t.Errorf("empty-state message missing from view:\n%s", view)
	}
}

func TestView_tinyTerminalDoesNotPanic(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/very-long-branch-name-for-narrow-terminals"))
	m = apply(t, m, tea.WindowSizeMsg{Width: 20, Height: 5})
	_ = m.View().Content // must not panic
}

// --- navigation ---

func TestUpdate_focusMovesAndClamps(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	if m.focus != 0 {
		t.Fatalf("initial focus = %d, want 0", m.focus)
	}
	m = apply(t, m, press('k'))
	if m.focus != 0 {
		t.Errorf("focus after k at top = %d, want clamped 0", m.focus)
	}
	m = apply(t, m, press('j'), press('j'))
	if m.focus != 2 {
		t.Errorf("focus after jj = %d, want 2", m.focus)
	}
	for range 20 {
		m = apply(t, m, press('j'))
	}
	if m.focus != len(m.rows)-1 {
		t.Errorf("focus after many j = %d, want %d (clamped)", m.focus, len(m.rows)-1)
	}
}

// --- collapse / expand ---

func TestUpdate_collapseGroupHidesChildren(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))
	// rows: repo, main-leaf, feature-group, a, b (main pinned first)
	var groupIdx int
	for i, r := range m.rows {
		if r.kind == rowGroup {
			groupIdx = i
			break
		}
	}
	before := len(m.rows)
	m.focus = groupIdx
	m = apply(t, m, press('h'))
	if len(m.rows) != before-2 {
		t.Errorf("rows after collapse = %d, want %d", len(m.rows), before-2)
	}
	m = apply(t, m, press('l'))
	if len(m.rows) != before {
		t.Errorf("rows after expand = %d, want %d", len(m.rows), before)
	}
}

func TestUpdate_collapseOnLeafJumpsToAncestor(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))
	var leafIdx int
	for i, r := range m.rows {
		if r.kind == rowLeaf && r.node.Name == "a" {
			leafIdx = i
			break
		}
	}
	m.focus = leafIdx
	m = apply(t, m, press('h'))
	row := m.rows[m.focus]
	if row.kind != rowGroup || row.node.Name != "feature" {
		t.Errorf("focus after h on leaf = %+v, want the feature group", row)
	}
	for _, r := range m.rows {
		if r.kind == rowLeaf && (r.node.Name == "a" || r.node.Name == "b") {
			t.Error("children should be hidden after ancestor collapse")
		}
	}
}

func TestUpdate_collapseOnRootLeafCollapsesRepoHeader(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	var mainIdx int
	for i, r := range m.rows {
		if r.kind == rowLeaf && r.node.Name == "main" {
			mainIdx = i
			break
		}
	}
	m.focus = mainIdx
	m = apply(t, m, press('h'))
	row := m.rows[m.focus]
	if row.kind != rowRepo {
		t.Errorf("focus after h on root leaf = %+v, want repo header", row)
	}
	if len(m.rows) != 1 {
		t.Errorf("rows after repo collapse = %d, want 1", len(m.rows))
	}
}

// --- filter / merged / sort ---

func TestUpdate_filterNarrowsRowsIncrementally(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/rate-limit", "chore/cleanup"))
	m = apply(t, m, press('/'))
	if !m.filtering {
		t.Fatal("/ should focus the filter input")
	}
	m = apply(t, m, press('r'), press('a'), press('t'), press('e'))
	for _, r := range m.rows {
		if r.kind == rowLeaf && strings.Contains(r.node.Name, "cleanup") {
			t.Error("non-matching leaf should be filtered out")
		}
	}
	m = apply(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.filtering {
		t.Error("Esc should leave the filter input")
	}
	if m.filter != "rate" {
		t.Errorf("filter = %q, want rate (kept after Esc)", m.filter)
	}
}

func TestUpdate_arrowsMoveFocusWhileFiltering(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/aa", "feature/ab"))
	m = apply(t, m, press('/'), press('a'))
	before := m.focus
	m = apply(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.focus != before+1 {
		t.Errorf("focus = %d, want %d (down arrow should move focus during filtering)", m.focus, before+1)
	}
	if !m.filtering {
		t.Error("arrow navigation should not leave the filter input")
	}
	m = apply(t, m, press('j'))
	if m.filter != "aj" {
		t.Errorf("filter = %q, want %q (j stays literal text)", m.filter, "aj")
	}
}

func TestUpdate_filterWithNoMatchesShowsEmptyState(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	m = apply(t, m, press('/'), press('z'), press('z'), press('z'))
	if len(m.rows) != 0 {
		t.Errorf("rows = %d, want 0 for a no-match filter", len(m.rows))
	}
	view := m.View().Content
	if !strings.Contains(view, "No worktrees") {
		t.Errorf("empty-state message missing:\n%s", view)
	}
}

func TestUpdate_mergedOnlyToggles(t *testing.T) {
	t.Parallel()
	repo := testRepo("alpha", "feature/keep")
	repo.Worktrees[1].Merged = true
	repo.Worktrees = append(repo.Worktrees, domain.Worktree{
		Path: "/work/alpha-wt/drop", Branch: "feature/drop",
		LastCommitTime: time.Unix(1_600_000_000, 0),
	})
	m := modelWith(t, repo)

	m = apply(t, m, press('m'))
	for _, r := range m.rows {
		if r.kind == rowLeaf && r.node.Worktree.Branch == "feature/drop" {
			t.Error("unmerged leaf visible under merged-only")
		}
	}
	m = apply(t, m, press('m'))
	found := false
	for _, r := range m.rows {
		if r.kind == rowLeaf && r.node.Worktree.Branch == "feature/drop" {
			found = true
		}
	}
	if !found {
		t.Error("second m should restore unmerged leaves")
	}
}

func TestUpdate_sortCyclesAndSwitchesToFlat(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))

	hasGroup := func(m Model) bool {
		for _, r := range m.rows {
			if r.kind == rowGroup {
				return true
			}
		}
		return false
	}
	if !hasGroup(m) {
		t.Fatal("tree mode should show groups")
	}
	m = apply(t, m, press('s')) // stale
	if m.sortMode != domain.StaleFirst || hasGroup(m) {
		t.Errorf("after s: mode=%v groups=%v, want StaleFirst and flat", m.sortMode, hasGroup(m))
	}
	m = apply(t, m, press('s'), press('s'), press('s')) // fresh, merged, tree
	if m.sortMode != domain.TreeView || !hasGroup(m) {
		t.Errorf("after ssss: mode=%v, want TreeView with groups back", m.sortMode)
	}
}

func TestUpdate_quitReturnsQuitCmd(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha"))
	var model tea.Model = m
	_, cmd := model.Update(press('q'))
	if cmd == nil {
		t.Fatal("q should produce a quit command")
	}
}

func TestView_flatModeShowsFullBranchNames(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	m = apply(t, m, press('s'))
	view := m.View().Content
	if !strings.Contains(view, "feature/x") {
		t.Errorf("flat view should show the full branch name:\n%s", view)
	}
}
