package domain

// SortMode selects how worktrees are ordered and whether the branch-name
// tree hierarchy is rendered (TreeView) or flattened (all other modes).
type SortMode int

const (
	// TreeView renders the slash-delimited branch hierarchy, sorted
	// alphabetically at each level.
	TreeView SortMode = iota
	// StaleFirst flattens the tree and sorts oldest last commit first.
	StaleFirst
	// FreshFirst flattens the tree and sorts newest last commit first.
	FreshFirst
	// MergedFirst flattens the tree and sorts merged branches first,
	// tie-broken by staleness (oldest first).
	MergedFirst
)

// Next returns the following mode in the cycle
// TreeView → StaleFirst → FreshFirst → MergedFirst → TreeView.
func (m SortMode) Next() SortMode {
	switch m {
	case TreeView:
		return StaleFirst
	case StaleFirst:
		return FreshFirst
	case FreshFirst:
		return MergedFirst
	case MergedFirst:
		return TreeView
	default:
		return TreeView
	}
}

// Flat reports whether this mode abandons the tree hierarchy and renders a
// flat per-repository list.
func (m SortMode) Flat() bool {
	return m != TreeView
}
