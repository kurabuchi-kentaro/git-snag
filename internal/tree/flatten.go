package tree

import (
	"strings"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// Row is one renderable line of the tree, with the sibling-position
// information the box-drawing connectors need.
type Row struct {
	Node *Node
	// IsLast[d] reports whether the ancestor at depth d — and, at the final
	// index, the node itself — is the last among its siblings. Depth 0 is a
	// direct child of the root. Rendering derives "│   " vs "    " from the
	// ancestor entries and "├── " vs "└── " from the final entry.
	IsLast []bool
}

// Depth returns the nesting depth of the row (0 = direct child of root).
func (r Row) Depth() int { return len(r.IsLast) - 1 }

// Flatten walks the tree in display order and returns the visible rows.
// Children of collapsed nodes are omitted (the row itself stays visible).
// Any node with children descends — groups, and the branch-mode root
// anchor, which is a leaf that carries the subtree (ADR 0014).
func Flatten(root *Node, collapsed map[string]bool) []Row {
	var rows []Row
	var walk func(n *Node, ancestry []bool)
	walk = func(n *Node, ancestry []bool) {
		for i, c := range n.Children {
			isLast := i == len(n.Children)-1
			path := append(append([]bool{}, ancestry...), isLast)
			rows = append(rows, Row{Node: c, IsLast: path})
			if len(c.Children) > 0 && !collapsed[c.ID] {
				walk(c, path)
			}
		}
	}
	walk(root, nil)
	return rows
}

// Filter returns the worktrees whose branch name or filesystem path contains
// query (case-insensitive), further restricted to merged worktrees when
// mergedOnly is set. Building a tree from the result naturally drops groups
// that lost all their leaves. An empty query matches everything.
func Filter(worktrees []domain.Worktree, query string, mergedOnly bool) []domain.Worktree {
	q := strings.ToLower(query)
	var out []domain.Worktree
	for _, w := range worktrees {
		if mergedOnly && !w.Merged {
			continue
		}
		if q != "" {
			haystack := strings.ToLower(w.Branch + " " + w.Path)
			if !strings.Contains(haystack, q) {
				continue
			}
		}
		out = append(out, w)
	}
	return out
}

// FilterBranches is Filter for branch mode. mergedOnly matches merged OR
// upstream-gone branches: in squash-merge workflows the gone state is the
// working definition of "merged" (ADR 0014).
func FilterBranches(branches []domain.Branch, query string, mergedOnly bool) []domain.Branch {
	q := strings.ToLower(query)
	var out []domain.Branch
	for _, b := range branches {
		if mergedOnly && !b.Merged && !b.UpstreamGone {
			continue
		}
		if q != "" {
			haystack := strings.ToLower(b.Name + " " + b.WorktreePath)
			if !strings.Contains(haystack, q) {
				continue
			}
		}
		out = append(out, b)
	}
	return out
}
