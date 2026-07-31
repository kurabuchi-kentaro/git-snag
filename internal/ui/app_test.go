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
		Path:          "/work/" + name,
		Name:          name,
		DefaultBranch: "main",
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
	// Mirror the enricher: every worktree branch exists as a local branch,
	// with the default branch protected.
	for _, w := range repo.Worktrees {
		repo.Branches = append(repo.Branches, domain.Branch{
			Name:           w.Branch,
			WorktreePath:   w.Path,
			LastCommitTime: w.LastCommitTime,
			Protected:      w.IsMain || w.Branch == repo.DefaultBranch,
		})
	}
	return repo
}

// withBareBranch appends a worktree-less local branch to the repo.
func withBareBranch(repo domain.Repo, name string, opts ...func(*domain.Branch)) domain.Repo {
	b := domain.Branch{Name: name, LastCommitTime: time.Unix(1_600_000_000, 0)}
	for _, o := range opts {
		o(&b)
	}
	repo.Branches = append(repo.Branches, b)
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

// --- info level ---

func TestUpdate_infoLevelCyclesThroughThreeStates(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	for i, want := range []int{1, 2, 0} {
		m = apply(t, m, press('i'))
		if m.infoLevel != want {
			t.Errorf("infoLevel after %d presses = %d, want %d", i+1, m.infoLevel, want)
		}
	}
}

func TestUpdate_infoLevelSkipsAllReposInFlatMode(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	m = apply(t, m, press('s')) // flat: the all-repos level changes nothing
	for i, want := range []int{2, 0, 2} {
		m = apply(t, m, press('i'))
		if m.infoLevel != want {
			t.Errorf("infoLevel after %d presses = %d, want %d (level 1 skipped)", i+1, m.infoLevel, want)
		}
	}
	s := stripANSI(m.statusLine())
	if strings.Contains(s, "all repos") {
		t.Errorf("flat-mode label should not claim all repos: %q", s)
	}
	if !strings.Contains(s, "paths") {
		t.Errorf("flat-mode label should mention paths at level 2: %q", s)
	}
}

func TestView_singleWorktreeRepoHiddenByDefault(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("solo"), testRepo("multi", "feature/x"))
	view := m.View().Content
	if strings.Contains(view, "solo") {
		t.Errorf("single-worktree repo should be hidden at info level 0:\n%s", view)
	}
	if !strings.Contains(view, "multi") {
		t.Errorf("multi-worktree repo must stay visible:\n%s", view)
	}
	m = apply(t, m, press('i'))
	if view := m.View().Content; !strings.Contains(view, "solo") {
		t.Errorf("i should reveal single-worktree repos:\n%s", view)
	}
}

func TestView_pathLineOnlyAtFullInfoLevel(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	if view := m.View().Content; strings.Contains(view, "alpha-wt/feature/x") {
		t.Errorf("worktree path should be hidden below info level 2:\n%s", view)
	}
	m = apply(t, m, press('i'), press('i'))
	if view := m.View().Content; !strings.Contains(view, "alpha-wt/feature/x") {
		t.Errorf("second i should reveal worktree paths:\n%s", view)
	}
}

func TestView_leafAnnotatesFullBranchName(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/x"))
	view := stripANSI(m.View().Content)
	if !strings.Contains(view, "x (feature/x)") {
		t.Errorf("leaf should carry its branch in parens:\n%s", view)
	}
	if !strings.Contains(view, "alpha (main)") {
		t.Errorf("main worktree leaf should carry its branch too:\n%s", view)
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

func TestView_flatModeHidesNonDeletableWorktrees(t *testing.T) {
	t.Parallel()
	repo := testRepo("alpha", "feature/x")
	repo.DefaultBranch = "main"
	m := modelWith(t, repo)

	m = apply(t, m, press('s')) // flat: deletion candidates only
	view := stripANSI(m.View().Content)
	if strings.Contains(view, "(main)") {
		t.Errorf("flat view should hide the main/default-branch worktree:\n%s", view)
	}

	m = apply(t, m, press('s'), press('s'), press('s')) // back to tree
	if view := stripANSI(m.View().Content); !strings.Contains(view, "(main)") {
		t.Errorf("tree view should keep the main worktree visible:\n%s", view)
	}
}

func TestView_flatModeSortsAcrossRepos(t *testing.T) {
	t.Parallel()
	oldRepo := testRepo("alpha", "feature/old")
	oldRepo.Worktrees[1].LastCommitTime = time.Unix(1_500_000_000, 0)
	freshRepo := testRepo("beta", "feature/new")
	freshRepo.Worktrees[1].LastCommitTime = time.Unix(1_650_000_000, 0)
	midRepo := testRepo("gamma", "feature/mid")
	midRepo.Worktrees[1].LastCommitTime = time.Unix(1_600_000_000, 0)
	m := modelWith(t, oldRepo, freshRepo, midRepo)

	m = apply(t, m, press('s')) // StaleFirst
	var branches []string
	for _, r := range m.rows {
		if r.kind == rowRepo {
			t.Error("global flat mode should have no repo headers")
		}
		if r.kind == rowLeaf {
			branches = append(branches, r.node.Worktree.Branch)
		}
	}
	want := []string{"feature/old", "feature/mid", "feature/new"}
	if len(branches) != len(want) {
		t.Fatalf("branches = %v, want %v", branches, want)
	}
	for i := range want {
		if branches[i] != want[i] {
			t.Fatalf("branches = %v, want %v (oldest first across repos)", branches, want)
		}
	}
	// Every row names its repository, since headers are gone.
	view := stripANSI(m.View().Content)
	for _, label := range []string{"/work/alpha ›", "/work/beta ›", "/work/gamma ›"} {
		if !strings.Contains(view, label) {
			t.Errorf("flat rows should carry the repo label %q:\n%s", label, view)
		}
	}
}

func TestView_flatModeDropsReposWithNoCandidates(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("solo"), testRepo("multi", "feature/x"))
	m = apply(t, m, press('i'), press('s')) // reveal solo, then go flat
	view := stripANSI(m.View().Content)
	if strings.Contains(view, "solo") {
		t.Errorf("a repo with no deletion candidates should vanish in flat mode:\n%s", view)
	}
}
