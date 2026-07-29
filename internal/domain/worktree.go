// Package domain holds the core data types shared across git-snag's layers.
// It performs no I/O.
package domain

import "time"

// Worktree describes a single git worktree and the status git-snag derives
// for it. Branch is empty when the worktree has a detached HEAD.
type Worktree struct {
	Path           string
	Branch         string
	HeadSHA        string
	IsMain         bool
	IsCurrent      bool
	Locked         bool
	Prunable       bool
	Dirty          bool
	Merged         bool
	UnpushedCount  int
	HasUpstream    bool
	LastCommitTime time.Time
}

// Protected reports whether git refuses to remove this worktree: the main
// worktree of a repository, or the worktree the user is currently inside.
func (w Worktree) Protected() bool {
	return w.IsMain || w.IsCurrent
}

// Detached reports whether the worktree has a detached HEAD (no branch).
func (w Worktree) Detached() bool {
	return w.Branch == ""
}
