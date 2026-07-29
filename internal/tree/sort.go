package tree

import (
	"sort"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// containsProtected reports whether the subtree holds any protected worktree.
// A group inherits the priority of its most prioritized descendant, so a
// subtree with the main/current worktree pins its whole group first.
func containsProtected(n *Node) bool {
	if !n.IsGroup() {
		return n.Worktree.Protected()
	}
	for _, c := range n.Children {
		if containsProtected(c) {
			return true
		}
	}
	return false
}

// sortTree orders every level: protected-containing subtrees first, then
// alphabetically by display name.
func sortTree(n *Node) {
	for _, c := range n.Children {
		if c.IsGroup() {
			sortTree(c)
		}
	}
	sort.SliceStable(n.Children, func(i, j int) bool {
		a, b := n.Children[i], n.Children[j]
		if pa, pb := containsProtected(a), containsProtected(b); pa != pb {
			return pa
		}
		return a.Name < b.Name
	})
}

// sortFlat orders leaves for the flat modes: protected pinned first, then
// the mode's own ordering.
func sortFlat(leaves []*Node, mode domain.SortMode) {
	sort.SliceStable(leaves, func(i, j int) bool {
		a, b := leaves[i].Worktree, leaves[j].Worktree
		if pa, pb := a.Protected(), b.Protected(); pa != pb {
			return pa
		}
		switch mode {
		case domain.FreshFirst:
			return a.LastCommitTime.After(b.LastCommitTime)
		case domain.MergedFirst:
			if a.Merged != b.Merged {
				return a.Merged
			}
			return a.LastCommitTime.Before(b.LastCommitTime)
		case domain.StaleFirst, domain.TreeView:
			return a.LastCommitTime.Before(b.LastCommitTime)
		default:
			return a.LastCommitTime.Before(b.LastCommitTime)
		}
	})
}
