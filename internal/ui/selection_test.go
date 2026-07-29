package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

func space() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
}

func esc() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEscape}
}

func focusOn(t *testing.T, m *Model, match func(uiRow) bool) {
	t.Helper()
	for i, r := range m.rows {
		if match(r) {
			m.focus = i
			return
		}
	}
	t.Fatal("no row matched")
}

func leafNamed(name string) func(uiRow) bool {
	return func(r uiRow) bool { return r.kind == rowLeaf && r.node.Name == name }
}

func groupNamed(name string) func(uiRow) bool {
	return func(r uiRow) bool { return r.kind == rowGroup && r.node.Name == name }
}

func selectedBranches(m Model) map[string]bool {
	out := map[string]bool{}
	for path := range m.selection {
		for i := range m.repos {
			for _, w := range m.repos[i].Worktrees {
				if w.Path == path {
					out[w.Branch] = true
				}
			}
		}
	}
	return out
}

// --- direct selection ---

func TestSpace_togglesLeafSelection(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space())
	if got := selectedBranches(m); !got["feature/a"] {
		t.Fatalf("selection = %v, want feature/a", got)
	}
	m = apply(t, m, space())
	if len(m.selection) != 0 {
		t.Errorf("second space should clear: %v", m.selection)
	}
}

func TestSpace_onProtectedLeafDoesNothing(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("main"))
	m = apply(t, m, space())
	if len(m.selection) != 0 {
		t.Errorf("protected leaf must not be selectable: %v", m.selection)
	}
}

func TestSpace_onGroupSelectsAllUnprotectedDescendants(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b", "feature/fix/c"))
	focusOn(t, &m, groupNamed("feature"))
	m = apply(t, m, space())
	got := selectedBranches(m)
	for _, b := range []string{"feature/a", "feature/b", "feature/fix/c"} {
		if !got[b] {
			t.Errorf("%s should be selected: %v", b, got)
		}
	}
	if got["main"] {
		t.Error("protected main must not be selected")
	}
	m = apply(t, m, space())
	if len(m.selection) != 0 {
		t.Errorf("second space should clear all: %v", m.selection)
	}
}

func TestSpace_onRepoHeaderSelectsWholeRepo(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "solo"))
	m.focus = 0 // repo header
	m = apply(t, m, space())
	got := selectedBranches(m)
	if !got["feature/a"] || !got["solo"] {
		t.Errorf("selection = %v, want both leaves", got)
	}
	if got["main"] {
		t.Error("protected main must not be selected")
	}
}

func TestSelectionState_triState(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))
	var group uiRow
	for _, r := range m.rows {
		if r.kind == rowGroup {
			group = r
		}
	}
	if got := m.selectionState(group); got != selNone {
		t.Errorf("state = %v, want selNone", got)
	}
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space())
	for _, r := range m.rows {
		if r.kind == rowGroup {
			group = r
		}
	}
	if got := m.selectionState(group); got != selPartial {
		t.Errorf("state = %v, want selPartial", got)
	}
	focusOn(t, &m, leafNamed("b"))
	m = apply(t, m, space())
	for _, r := range m.rows {
		if r.kind == rowGroup {
			group = r
		}
	}
	if got := m.selectionState(group); got != selAll {
		t.Errorf("state = %v, want selAll", got)
	}
}

// --- filter interplay (REQ-A9 / REQ-A10) ---

func TestSpace_onGroupUnderFilterSelectsOnlyVisibleLeaves(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/apple", "feature/banana"))
	m = apply(t, m, press('/'), press('a'), press('p'), press('p'), esc())
	focusOn(t, &m, groupNamed("feature"))
	m = apply(t, m, space())
	got := selectedBranches(m)
	if !got["feature/apple"] || got["feature/banana"] {
		t.Errorf("selection = %v, want only the visible feature/apple", got)
	}
}

func TestSelection_persistsAcrossFilterAndSortChanges(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/keep", "feature/other"))
	focusOn(t, &m, leafNamed("keep"))
	m = apply(t, m, space())

	m = apply(t, m, press('/'), press('z'), press('z'), esc()) // hides everything
	if got := selectedBranches(m); !got["feature/keep"] {
		t.Errorf("selection lost while filtered out: %v", got)
	}
	m.filterInput.SetValue("")
	m = apply(t, m, press('/'), esc(), press('s'), press('s'), press('s'), press('s'))
	if got := selectedBranches(m); !got["feature/keep"] {
		t.Errorf("selection lost across sort cycling: %v", got)
	}
}

// --- Visual mode ---

func TestVisual_confirmAddsRangeToExistingSelection(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b", "feature/c"))
	// Pre-select c so the range is additive on top.
	focusOn(t, &m, leafNamed("c"))
	m = apply(t, m, space())
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, press('v'), press('j'), press('v'))
	got := selectedBranches(m)
	for _, b := range []string{"feature/a", "feature/b", "feature/c"} {
		if !got[b] {
			t.Errorf("%s should be selected: %v", b, got)
		}
	}
	if m.visualAnchor != -1 {
		t.Error("v should confirm and leave visual mode")
	}
}

func TestVisual_escapeRevertsToPreVisualSelection(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b", "feature/c"))
	focusOn(t, &m, leafNamed("c"))
	m = apply(t, m, space())
	before := selectedBranches(m)
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, press('v'), press('j'), esc())
	after := selectedBranches(m)
	if len(after) != len(before) || !after["feature/c"] {
		t.Errorf("selection after Esc = %v, want exactly %v", after, before)
	}
	if m.visualAnchor != -1 {
		t.Error("Esc should leave visual mode")
	}
}

func TestVisual_rangeSkipsProtectedAndGroupRows(t *testing.T) {
	t.Parallel()
	// Tree order: repo, main(protected), feature-group, a, b.
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))
	m.focus = 0
	m = apply(t, m, press('v'))
	for range len(m.rows) - 1 {
		m = apply(t, m, press('j'))
	}
	m = apply(t, m, press('v'))
	got := selectedBranches(m)
	if got["main"] {
		t.Error("protected leaf inside range must not be selected")
	}
	if !got["feature/a"] || !got["feature/b"] {
		t.Errorf("selectable leaves in range should be selected: %v", got)
	}
}

func TestVisual_ignoresFilterMergedSortAndDelete(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, press('v'), press('/'), press('m'), press('s'))
	if !m.filtering == false || m.mergedOnly || m.sortMode != domain.TreeView {
		t.Errorf("visual mode must ignore /, m, s: filtering=%v merged=%v sort=%v",
			m.filtering, m.mergedOnly, m.sortMode)
	}
	if m.visualAnchor == -1 {
		t.Error("still in visual mode after ignored keys")
	}
}

func TestVisual_mouseClickConfirmsAndExits(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, press('v'), press('j'))
	m = apply(t, m, tea.MouseClickMsg{})
	if m.visualAnchor != -1 {
		t.Error("mouse click should confirm-exit visual mode")
	}
	if got := selectedBranches(m); !got["feature/a"] {
		t.Errorf("range selection should be kept: %v", got)
	}
}

func TestEscape_outsideVisualClearsSelection(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space(), esc())
	if len(m.selection) != 0 {
		t.Errorf("Esc should clear selection: %v", m.selection)
	}
}

func TestFooter_showsSelectionCount(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space())
	view := m.View().Content
	if !strings.Contains(view, "1 selected") {
		t.Errorf("view should mention selection count:\n%s", view)
	}
}
