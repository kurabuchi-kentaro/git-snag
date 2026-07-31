package tree

import (
	"sort"
	"time"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// leafMeta returns the sort-relevant status of a leaf, whichever payload it
// carries.
func leafMeta(n *Node) (protected, merged bool, t time.Time) {
	switch {
	case n.Worktree != nil:
		return n.Worktree.Protected(), n.Worktree.Merged, n.Worktree.LastCommitTime
	case n.Branch != nil:
		return n.Branch.Protected, n.Branch.Merged, n.Branch.LastCommitTime
	default:
		return false, false, time.Time{}
	}
}

// containsProtected reports whether the subtree holds any protected leaf.
// A group inherits the priority of its most prioritized descendant, so a
// subtree with the main/current worktree pins its whole group first.
func containsProtected(n *Node) bool {
	if !n.IsGroup() {
		p, _, _ := leafMeta(n)
		return p
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
		aProt, aMerged, aTime := leafMeta(leaves[i])
		bProt, bMerged, bTime := leafMeta(leaves[j])
		if aProt != bProt {
			return aProt
		}
		switch mode {
		case domain.FreshFirst:
			return aTime.After(bTime)
		case domain.MergedFirst:
			if aMerged != bMerged {
				return aMerged
			}
			return aTime.Before(bTime)
		case domain.StaleFirst, domain.TreeView:
			return aTime.Before(bTime)
		default:
			return aTime.Before(bTime)
		}
	})
}
