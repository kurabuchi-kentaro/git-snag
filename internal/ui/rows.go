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

// branchKey identifies a branch row (and its selection entry) across
// repositories. Branch names and group IDs cannot collide within a repo:
// git's ref namespacing forbids a branch "feature" next to "feature/x".
func branchKey(repoPath, branch string) string {
	return repoPath + "\x00" + branch
}

// rebuildRows recomputes the visible row list from the model's repos, view
// mode, filter, sort mode, and collapse state.
func (m *Model) rebuildRows() {
	m.rows = m.rows[:0]
	if m.branchMode {
		m.wtByPath = map[string]*domain.Worktree{}
		for i := range m.repos {
			for j := range m.repos[i].Worktrees {
				w := &m.repos[i].Worktrees[j]
				m.wtByPath[w.Path] = w
			}
		}
	}
	switch {
	case m.branchMode && m.sortMode.Flat():
		m.appendGlobalFlatBranchRows()
	case m.branchMode:
		m.appendBranchTreeRows()
	case m.sortMode.Flat():
		m.appendGlobalFlatRows()
	default:
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

// bareCollapsedGroups translates the repo-namespaced collapsed-group keys
// down to the bare node IDs tree.Flatten expects.
func (m *Model) bareCollapsedGroups(repoPath string) map[string]bool {
	bare := map[string]bool{}
	for id := range m.collapsedGroups {
		if p, gid, ok := splitGroupKey(id); ok && p == repoPath {
			bare[gid] = true
		}
	}
	return bare
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
		root := tree.Build(visible)
		// The header row carries the repo's whole tree so selection
		// gestures and the tri-state mark reuse it instead of rebuilding.
		m.rows = append(m.rows, uiRow{
			kind:    rowRepo,
			repoIdx: idx,
			key:     repo.Path,
			node:    root,
		})
		if m.collapsedRepos[repo.Path] {
			continue
		}

		parents := []string{repo.Path}
		for _, r := range tree.Flatten(root, m.bareCollapsedGroups(repo.Path)) {
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
			row.parentKey, parents = parentFromDepth(parents, r.Depth(), row.key)
			m.rows = append(m.rows, row)
		}
	}
}

// parentFromDepth resolves a row's collapse parent from the flattened walk
// itself — the nearest enclosing row, or the repo header at depth 0 — and
// records the row as the candidate parent for the next depth. Deriving the
// parent from names would point at groups that single-child compaction
// folded away.
func parentFromDepth(parents []string, depth int, key string) (string, []string) {
	parent := parents[min(depth, len(parents)-1)]
	parents = append(parents[:min(depth+1, len(parents))], key)
	return parent, parents
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

// appendBranchTreeRows renders branch mode's TreeView (ADR 0014): a header
// per repository with the local branches nested beneath it by
// slash-delimited name, the muted default branch pinned first.
func (m *Model) appendBranchTreeRows() {
	for idx := range m.repos {
		repo := &m.repos[idx]
		visible, ok := m.repoBranches(repo)
		if !ok {
			continue
		}
		root := tree.BuildBranches(visible)
		// The header row carries the repo's whole tree so selection
		// gestures and the tri-state mark reuse it instead of rebuilding.
		m.rows = append(m.rows, uiRow{
			kind:    rowRepo,
			repoIdx: idx,
			key:     repo.Path,
			node:    root,
		})
		if m.collapsedRepos[repo.Path] {
			continue
		}

		parents := []string{repo.Path}
		for _, r := range tree.Flatten(root, m.bareCollapsedGroups(repo.Path)) {
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
				row.key = branchKey(repo.Path, r.Node.ID)
			}
			row.parentKey, parents = parentFromDepth(parents, r.Depth(), row.key)
			m.rows = append(m.rows, row)
		}
	}
}

// appendGlobalFlatBranchRows renders branch mode's flat sort modes as one
// repo-spanning candidate list, mirroring appendGlobalFlatRows.
func (m *Model) appendGlobalFlatBranchRows() {
	var all []domain.Branch
	var repoIdxs []int
	for idx := range m.repos {
		visible, ok := m.repoBranches(&m.repos[idx])
		if !ok {
			continue
		}
		for _, b := range visible {
			all = append(all, b)
			repoIdxs = append(repoIdxs, idx)
		}
	}
	root := tree.BuildFlatBranches(all, m.sortMode)
	byPtr := map[*domain.Branch]int{}
	for i := range all {
		byPtr[&all[i]] = repoIdxs[i]
	}
	for _, r := range tree.Flatten(root, nil) {
		idx := byPtr[r.Node.Branch]
		m.rows = append(m.rows, uiRow{
			kind:    rowLeaf,
			repoIdx: idx,
			key:     branchKey(m.repos[idx].Path, r.Node.ID),
			node:    r.Node,
			isLast:  r.IsLast,
			flat:    true,
		})
	}
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

// visibleBranches is visibleWorktrees for branch mode: the text/merged-or-
// gone filter, and in flat sort modes only deletable (non-protected)
// branches remain.
func (m *Model) visibleBranches(repo *domain.Repo) []domain.Branch {
	visible := tree.FilterBranches(repo.Branches, m.filter, m.mergedOnly)
	if !m.sortMode.Flat() {
		return visible
	}
	kept := visible[:0]
	for _, b := range visible {
		if b.Protected {
			continue
		}
		kept = append(kept, b)
	}
	return kept
}

// repoBranches is repoWorktrees for branch mode: ok=false hides repos that
// contribute nothing — no deletable branch at info level 0, or emptied by
// the filter, merged-or-gone toggle, or the flat modes' candidates-only
// rule.
func (m *Model) repoBranches(repo *domain.Repo) ([]domain.Branch, bool) {
	if m.infoLevel == 0 {
		deletable := 0
		for _, b := range repo.Branches {
			if !b.Protected {
				deletable++
			}
		}
		if deletable == 0 {
			return nil, false
		}
	}
	visible := m.visibleBranches(repo)
	if (m.filter != "" || m.mergedOnly || m.sortMode.Flat()) && len(visible) == 0 {
		return nil, false
	}
	return visible, true
}

// worktreeByPath returns the repo's worktree at path, or nil.
func worktreeByPath(repo *domain.Repo, path string) *domain.Worktree {
	if path == "" {
		return nil
	}
	for i := range repo.Worktrees {
		if repo.Worktrees[i].Path == path {
			return &repo.Worktrees[i]
		}
	}
	return nil
}

// branchByName returns the repo's branch with the given name, or nil.
func branchByName(repo *domain.Repo, name string) *domain.Branch {
	for i := range repo.Branches {
		if repo.Branches[i].Name == name {
			return &repo.Branches[i]
		}
	}
	return nil
}

// repoByPath returns the repo at path, or nil.
func repoByPath(repos []domain.Repo, path string) *domain.Repo {
	for i := range repos {
		if repos[i].Path == path {
			return &repos[i]
		}
	}
	return nil
}
