package domain

// Repo is a git repository discovered by the scanner, together with its
// worktrees, local branches, and the default branch used for merge
// detection. DefaultBranch is empty when it could not be determined (e.g.
// no origin remote).
type Repo struct {
	Path          string
	Name          string
	DefaultBranch string
	Worktrees     []Worktree
	Branches      []Branch
}
