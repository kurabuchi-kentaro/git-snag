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
// Children of groups whose ID is in collapsed are omitted (the group row
// itself stays visible).
func Flatten(root *Node, collapsed map[string]bool) []Row {
	var rows []Row
	var walk func(n *Node, ancestry []bool)
	walk = func(n *Node, ancestry []bool) {
		for i, c := range n.Children {
			isLast := i == len(n.Children)-1
			path := append(append([]bool{}, ancestry...), isLast)
			rows = append(rows, Row{Node: c, IsLast: path})
			if c.IsGroup() && !collapsed[c.ID] {
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
