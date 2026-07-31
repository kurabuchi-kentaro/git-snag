// Package tree turns a repository's worktrees into the displayed hierarchy:
// branch names split on "/" become nested groups (TreeView mode), or a flat
// sorted list (all other sort modes). It performs no I/O.
package tree

import (
	"fmt"
	"strings"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// Node is one element of the display tree. Groups have Children and neither
// leaf payload; leaves carry the Worktree or Branch they represent. The
// branch-mode root anchor is the one leaf that also has Children (ADR 0014).
type Node struct {
	// Name is the display label: the path segment for groups, the last
	// branch segment (or a detached-HEAD label) for tree-mode leaves, and
	// the full branch name for flat-mode leaves.
	Name string
	// ID uniquely identifies the node: the joined group path for groups,
	// the worktree's filesystem path for worktree leaves, the branch name
	// for branch leaves.
	ID       string
	Children []*Node
	Worktree *domain.Worktree
	Branch   *domain.Branch
}

// IsGroup reports whether the node is a synthetic directory-style group.
func (n *Node) IsGroup() bool { return n.Worktree == nil && n.Branch == nil }

// leafName returns the label of a leaf in tree mode.
func leafName(w domain.Worktree) string {
	if w.Detached() {
		sha := w.HeadSHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		return fmt.Sprintf("(detached: %s)", sha)
	}
	segments := strings.Split(w.Branch, "/")
	return segments[len(segments)-1]
}

// Build constructs the TreeView hierarchy: worktrees nested by their
// slash-delimited branch prefixes, sorted alphabetically per level with
// protected-containing subtrees pinned first. Group nodes are always
// synthetic: git's ref namespacing makes a branch named "feature" and one
// named "feature/x" mutually exclusive.
func Build(worktrees []domain.Worktree) *Node {
	root := &Node{}
	for i := range worktrees {
		w := worktrees[i]
		node := root
		if !w.Detached() {
			segments := strings.Split(w.Branch, "/")
			for _, seg := range segments[:len(segments)-1] {
				node = node.childGroup(seg)
			}
		}
		node.Children = append(node.Children, &Node{
			Name:     leafName(w),
			ID:       w.Path,
			Worktree: &worktrees[i],
		})
	}
	compactSingles(root)
	sortTree(root)
	return root
}

// compactSingles folds every group with exactly one child into that child,
// prefixing the group's name (VS Code's compact-folders convention): a lone
// feature/uc1-backend renders as one row instead of spending a row and an
// indent level on a feature/ group that disambiguates nothing. Bottom-up,
// so whole single-child chains collapse into a single label.
func compactSingles(n *Node) {
	for i, c := range n.Children {
		if !c.IsGroup() {
			continue
		}
		compactSingles(c)
		if len(c.Children) == 1 {
			child := c.Children[0]
			child.Name = c.Name + "/" + child.Name
			n.Children[i] = child
		}
	}
}

// BuildFlat constructs the flat rendering used by non-tree sort modes: every
// worktree is a direct child of the root, labeled with its full branch name,
// ordered by mode with protected worktrees pinned first.
func BuildFlat(worktrees []domain.Worktree, mode domain.SortMode) *Node {
	root := &Node{}
	for i := range worktrees {
		w := worktrees[i]
		name := w.Branch
		if w.Detached() {
			name = leafName(w)
		}
		root.Children = append(root.Children, &Node{
			Name:     name,
			ID:       w.Path,
			Worktree: &worktrees[i],
		})
	}
	sortFlat(root.Children, mode)
	return root
}

// BuildBranches constructs the branch-mode hierarchy (ADR 0014): branches
// nested by their slash-delimited name prefixes, exactly like Build. The
// protected default branch pins first through the shared sorting, mirroring
// the worktree tree — nesting everything under it would wrongly read as an
// ancestry claim, which branch names cannot make.
func BuildBranches(branches []domain.Branch) *Node {
	root := &Node{}
	for i := range branches {
		b := branches[i]
		node := root
		segments := strings.Split(b.Name, "/")
		for _, seg := range segments[:len(segments)-1] {
			node = node.childGroup(seg)
		}
		node.Children = append(node.Children, &Node{
			Name:   segments[len(segments)-1],
			ID:     b.Name,
			Branch: &branches[i],
		})
	}
	compactSingles(root)
	sortTree(root)
	return root
}

// BuildFlatBranches constructs the flat branch rendering used by non-tree
// sort modes: every branch is a direct child of the root, labeled with its
// full name, ordered by mode.
func BuildFlatBranches(branches []domain.Branch, mode domain.SortMode) *Node {
	root := &Node{}
	for i := range branches {
		root.Children = append(root.Children, &Node{
			Name:   branches[i].Name,
			ID:     branches[i].Name,
			Branch: &branches[i],
		})
	}
	sortFlat(root.Children, mode)
	return root
}

// childGroup returns the existing group child with the given name, creating
// it when absent.
func (n *Node) childGroup(name string) *Node {
	for _, c := range n.Children {
		if c.IsGroup() && c.Name == name {
			return c
		}
	}
	id := name
	if n.ID != "" {
		id = n.ID + "/" + name
	}
	g := &Node{Name: name, ID: id}
	n.Children = append(n.Children, g)
	return g
}
