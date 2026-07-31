package ui

import "github.com/kurabuchi-kentaro/git-snag/internal/tree"

// selState is a group/repo tri-state selection indicator.
type selState int

const (
	selNone selState = iota
	selPartial
	selAll
)

// selectableKeys collects the selection keys of every non-protected leaf in
// the subtree: worktree paths, or repo-namespaced branch keys. The tree is
// built from the filtered items, so this is exactly the set of currently
// *visible* leaves (REQ-A9); collapse state is a display fold and
// deliberately does not narrow it. The branch-mode root anchor is a leaf
// with children, so leaves recurse too.
func selectableKeys(repoPath string, n *tree.Node) []string {
	var out []string
	switch {
	case n.Worktree != nil:
		if !n.Worktree.Protected() {
			out = append(out, n.Worktree.Path)
		}
	case n.Branch != nil:
		if !n.Branch.Protected {
			out = append(out, branchKey(repoPath, n.Branch.Name))
		}
	}
	for _, c := range n.Children {
		out = append(out, selectableKeys(repoPath, c)...)
	}
	return out
}

// subtreeFor returns the node a selection gesture acts on. Every row —
// including repo headers, which carry their repo's whole tree — stores its
// node at rebuild time; building here is a fallback for safety.
func (m *Model) subtreeFor(row uiRow) *tree.Node {
	if row.node != nil {
		return row.node
	}
	repo := &m.repos[row.repoIdx]
	if m.branchMode {
		return tree.BuildBranches(m.visibleBranches(repo))
	}
	return tree.Build(m.visibleWorktrees(repo))
}

// toggleSelection implements space: a leaf toggles itself; a group or repo
// header toggles all its selectable descendants (all-or-nothing).
func (m *Model) toggleSelection(row uiRow) {
	paths := selectableKeys(m.repos[row.repoIdx].Path, m.subtreeFor(row))
	if len(paths) == 0 {
		return
	}
	all := true
	for _, p := range paths {
		if !m.selection[p] {
			all = false
			break
		}
	}
	for _, p := range paths {
		if all {
			delete(m.selection, p)
		} else {
			m.selection[p] = true
		}
	}
}

// selectionState reports the tri-state indicator for a group or repo row.
func (m *Model) selectionState(row uiRow) selState {
	paths := selectableKeys(m.repos[row.repoIdx].Path, m.subtreeFor(row))
	if len(paths) == 0 {
		return selNone
	}
	selected := 0
	for _, p := range paths {
		if m.selection[p] {
			selected++
		}
	}
	switch selected {
	case 0:
		return selNone
	case len(paths):
		return selAll
	default:
		return selPartial
	}
}

// enterVisual anchors visual mode at the focused row (REQ: vim-style range
// selection; see ADR 0006).
func (m *Model) enterVisual() {
	m.visualAnchor = m.focus
	m.preVisual = make(map[string]bool, len(m.selection))
	for p := range m.selection {
		m.preVisual[p] = true
	}
}

// applyVisualRange recomputes the live selection: everything selected before
// visual mode, plus every selectable leaf row between anchor and cursor.
// Group and header rows inside the range are skipped as rows — they never
// bulk-select their descendants here.
func (m *Model) applyVisualRange() {
	m.selection = make(map[string]bool, len(m.preVisual))
	for p := range m.preVisual {
		m.selection[p] = true
	}
	lo, hi := m.visualAnchor, m.focus
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo; i <= hi && i < len(m.rows); i++ {
		row := m.rows[i]
		if row.kind != rowLeaf {
			continue
		}
		// A leaf's row key is its selection key in both modes: the
		// worktree path, or the repo-namespaced branch key.
		n := row.node
		protected := (n.Worktree != nil && n.Worktree.Protected()) ||
			(n.Branch != nil && n.Branch.Protected)
		if !protected {
			m.selection[row.key] = true
		}
	}
}

// confirmVisual keeps the range selection and leaves visual mode.
func (m *Model) confirmVisual() {
	m.visualAnchor = -1
	m.preVisual = nil
}

// revertVisual restores the pre-visual selection and leaves visual mode.
func (m *Model) revertVisual() {
	m.selection = m.preVisual
	if m.selection == nil {
		m.selection = map[string]bool{}
	}
	m.visualAnchor = -1
	m.preVisual = nil
}
