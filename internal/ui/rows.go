package ui

import (
	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
	"github.com/kurabuchi-kentaro/git-snag/internal/tree"
)

type rowKind int

const (
	rowRepo rowKind = iota
	rowGroup
	rowLeaf
)

// uiRow is one focusable line group in the tree pane.
type uiRow struct {
	kind    rowKind
	repoIdx int
	// key uniquely identifies the row for collapse bookkeeping:
	// repo path for headers, repo path + "\x00" + group ID for groups,
	// worktree path for leaves.
	key string
	// parentKey is the key of the enclosing group or repo header; empty for
	// repo headers themselves. `h` on a leaf collapses this.
	parentKey string
	node      *tree.Node
	isLast    []bool
	flat      bool
}

// groupKey namespaces a group ID by its repository.
func groupKey(repoPath, groupID string) string {
	return repoPath + "\x00" + groupID
}

// rebuildRows recomputes the visible row list from the model's repos,
// filter, sort mode, and collapse state.
func (m *Model) rebuildRows() {
	m.rows = m.rows[:0]
	for idx := range m.repos {
		repo := &m.repos[idx]
		visible := tree.Filter(repo.Worktrees, m.filter, m.mergedOnly)
		filtering := m.filter != "" || m.mergedOnly
		if filtering && len(visible) == 0 {
			continue
		}
		m.rows = append(m.rows, uiRow{
			kind:    rowRepo,
			repoIdx: idx,
			key:     repo.Path,
		})
		if m.collapsedRepos[repo.Path] {
			continue
		}

		var root *tree.Node
		if m.sortMode.Flat() {
			root = tree.BuildFlat(visible, m.sortMode)
		} else {
			root = tree.Build(visible)
		}
		// tree.Flatten keys collapse state by node ID; translate the
		// repo-namespaced keys down to bare IDs for this repo.
		bare := map[string]bool{}
		for id := range m.collapsedGroups {
			if repoPath, gid, ok := splitGroupKey(id); ok && repoPath == repo.Path {
				bare[gid] = true
			}
		}
		for _, r := range tree.Flatten(root, bare) {
			row := uiRow{
				repoIdx: idx,
				node:    r.Node,
				isLast:  r.IsLast,
				flat:    m.sortMode.Flat(),
			}
			if r.Node.IsGroup() {
				row.kind = rowGroup
				row.key = groupKey(repo.Path, r.Node.ID)
			} else {
				row.kind = rowLeaf
				row.key = r.Node.ID
			}
			if row.flat {
				// Flat mode has no groups: everything collapses to the header.
				row.parentKey = repo.Path
			} else {
				row.parentKey = m.parentKeyFor(repo.Path, r.Node)
			}
			m.rows = append(m.rows, row)
		}
	}
	if m.focus >= len(m.rows) {
		m.focus = len(m.rows) - 1
	}
	if m.focus < 0 {
		m.focus = 0
	}
	m.ensureFocusVisible()
}

// parentKeyFor derives the collapse target of a node: its parent group when
// nested, otherwise the repository header.
func (m *Model) parentKeyFor(repoPath string, n *tree.Node) string {
	if n.IsGroup() {
		if i := lastIndexByte(n.ID, '/'); i >= 0 {
			return groupKey(repoPath, n.ID[:i])
		}
		return repoPath
	}
	if n.Worktree != nil && !n.Worktree.Detached() {
		branch := n.Worktree.Branch
		if i := lastIndexByte(branch, '/'); i >= 0 {
			return groupKey(repoPath, branch[:i])
		}
	}
	return repoPath
}

func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func splitGroupKey(key string) (repoPath, groupID string, ok bool) {
	for i := range key {
		if key[i] == 0 {
			return key[:i], key[i+1:], true
		}
	}
	return "", "", false
}

// visibleWorktreeCount counts a repository's worktrees after filtering.
func (m *Model) visibleWorktreeCount(repo *domain.Repo) int {
	return len(tree.Filter(repo.Worktrees, m.filter, m.mergedOnly))
}
