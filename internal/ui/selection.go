package ui

import "github.com/kurabuchi-kentaro/git-snag/internal/tree"

// selState is a group/repo tri-state selection indicator.
type selState int

const (
	selNone selState = iota
	selPartial
	selAll
)

// selectablePaths collects the worktree paths of every non-protected leaf in
// the subtree. The tree is built from the filtered worktrees, so this is
// exactly the set of currently *visible* leaves (REQ-A9); collapse state is
// a display fold and deliberately does not narrow it.
func selectablePaths(n *tree.Node) []string {
	if !n.IsGroup() {
		if n.Worktree.Protected() {
			return nil
		}
		return []string{n.Worktree.Path}
	}
	var out []string
	for _, c := range n.Children {
		out = append(out, selectablePaths(c)...)
	}
	return out
}

// subtreeFor returns the node a selection gesture acts on: the row's own
// node, or the repository's whole (filtered) tree for a header row.
// Headers only exist in tree mode — the global flat list has none.
func (m *Model) subtreeFor(row uiRow) *tree.Node {
	if row.node != nil {
		return row.node
	}
	return tree.Build(m.visibleWorktrees(&m.repos[row.repoIdx]))
}

// toggleSelection implements space: a leaf toggles itself; a group or repo
// header toggles all its selectable descendants (all-or-nothing).
func (m *Model) toggleSelection(row uiRow) {
	paths := selectablePaths(m.subtreeFor(row))
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
	paths := selectablePaths(m.subtreeFor(row))
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
		if row.kind == rowLeaf && !row.node.Worktree.Protected() {
			m.selection[row.node.Worktree.Path] = true
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
