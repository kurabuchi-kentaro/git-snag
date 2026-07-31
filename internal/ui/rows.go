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
	if m.sortMode.Flat() {
		m.appendGlobalFlatRows()
	} else {
		m.appendTreeRows()
	}
	if m.focus >= len(m.rows) {
		m.focus = len(m.rows) - 1
	}
	if m.focus < 0 {
		m.focus = 0
	}
	m.ensureFocusVisible()
}

// repoWorktrees returns a repository's displayed worktrees, reporting
// ok=false when the repo contributes nothing to the current view: hidden as
// a single-worktree repo at info level 0 (nothing deletable there), or
// emptied by the filter, the merged-only toggle, or the flat modes'
// candidates-only rule. Shared by the row builders and the status line so
// their counts always agree.
func (m *Model) repoWorktrees(repo *domain.Repo) ([]domain.Worktree, bool) {
	if m.infoLevel == 0 && len(repo.Worktrees) <= 1 {
		return nil, false
	}
	visible := m.visibleWorktrees(repo)
	if (m.filter != "" || m.mergedOnly || m.sortMode.Flat()) && len(visible) == 0 {
		return nil, false
	}
	return visible, true
}

// appendTreeRows renders TreeView mode: a header per repository with the
// branch hierarchy nested beneath it.
func (m *Model) appendTreeRows() {
	for idx := range m.repos {
		repo := &m.repos[idx]
		visible, ok := m.repoWorktrees(repo)
		if !ok {
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

		// tree.Flatten keys collapse state by node ID; translate the
		// repo-namespaced keys down to bare IDs for this repo.
		bare := map[string]bool{}
		for id := range m.collapsedGroups {
			if repoPath, gid, ok := splitGroupKey(id); ok && repoPath == repo.Path {
				bare[gid] = true
			}
		}
		for _, r := range tree.Flatten(tree.Build(visible), bare) {
			row := uiRow{
				repoIdx: idx,
				node:    r.Node,
				isLast:  r.IsLast,
			}
			if r.Node.IsGroup() {
				row.kind = rowGroup
				row.key = groupKey(repo.Path, r.Node.ID)
			} else {
				row.kind = rowLeaf
				row.key = r.Node.ID
			}
			row.parentKey = m.parentKeyFor(repo.Path, r.Node)
			m.rows = append(m.rows, row)
		}
	}
}

// appendGlobalFlatRows renders the flat sort modes as one repo-spanning
// list: ordering by staleness across repositories is the point of these
// modes, so repo headers disappear and each row carries its repository as
// a label instead.
func (m *Model) appendGlobalFlatRows() {
	var all []domain.Worktree
	repoIdx := map[string]int{}
	for idx := range m.repos {
		visible, ok := m.repoWorktrees(&m.repos[idx])
		if !ok {
			continue
		}
		for _, w := range visible {
			repoIdx[w.Path] = idx
			all = append(all, w)
		}
	}
	for _, r := range tree.Flatten(tree.BuildFlat(all, m.sortMode), nil) {
		m.rows = append(m.rows, uiRow{
			kind:    rowLeaf,
			repoIdx: repoIdx[r.Node.ID],
			key:     r.Node.ID,
			node:    r.Node,
			isLast:  r.IsLast,
			flat:    true,
		})
	}
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

// visibleWorktrees returns a repository's worktrees after the text/merged
// filter and, in flat sort modes, after dropping non-deletable entries
// (the main worktree and the default branch): flat modes line up deletion
// candidates, so infrastructure rows are pure noise there. The tree view
// keeps them — they anchor the hierarchy.
func (m *Model) visibleWorktrees(repo *domain.Repo) []domain.Worktree {
	visible := tree.Filter(repo.Worktrees, m.filter, m.mergedOnly)
	if !m.sortMode.Flat() {
		return visible
	}
	kept := visible[:0]
	for _, w := range visible {
		if w.IsMain || (w.Branch != "" && w.Branch == repo.DefaultBranch) {
			continue
		}
		kept = append(kept, w)
	}
	return kept
}

// visibleWorktreeCount counts a repository's worktrees after filtering.
func (m *Model) visibleWorktreeCount(repo *domain.Repo) int {
	return len(m.visibleWorktrees(repo))
}
