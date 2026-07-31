package domain

import "time"

// Branch describes a local branch and the status git-snag derives for it
// (ADR 0014). WorktreePath is empty when no worktree has the branch checked
// out.
type Branch struct {
	Name          string
	Merged        bool
	UnpushedCount int
	// UpstreamGone reports a configured upstream whose remote branch no
	// longer exists — the working "merged" signal in squash-merge flows.
	UpstreamGone   bool
	LastCommitTime time.Time
	WorktreePath   string
	// Protected marks branches that cannot be deleted: the default branch,
	// and branches checked out in a protected worktree (ADR 0010, 0014).
	// Set by the enricher.
	Protected bool
}

// HasWorktree reports whether the branch is checked out in some worktree.
func (b Branch) HasWorktree() bool {
	return b.WorktreePath != ""
}
