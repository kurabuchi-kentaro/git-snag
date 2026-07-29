package scan

import (
	"os"
	"path/filepath"
	"strings"
)

// gitDirKind classifies what the .git entry inside a directory means.
type gitDirKind int

const (
	notGit gitDirKind = iota
	// repository: .git is a directory — a standalone repository root.
	repository
	// linkedWorktree: .git is a file whose gitdir points into
	// <repo>/.git/worktrees/... — a linked worktree of some repository.
	linkedWorktree
	// submodule: .git is a file whose gitdir points into
	// <repo>/.git/modules/... — a submodule working tree.
	submodule
)

// classifyDir inspects dir's .git entry. A .git file is only ever written by
// git for linked worktrees and submodules; the gitdir path inside it tells
// the two apart (REQ-P3).
func classifyDir(dir string) gitDirKind {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return notGit
	}
	if info.IsDir() {
		return repository
	}
	if !info.Mode().IsRegular() {
		return notGit
	}
	// #nosec G304 -- gitPath is <walked dir>/.git, derived from the scan
	// walk, not from user-controlled input beyond the scan root itself.
	content, err := os.ReadFile(gitPath)
	if err != nil {
		return notGit
	}
	gitdir := strings.TrimSpace(strings.TrimPrefix(string(content), "gitdir:"))
	// Normalize separators so the check also behaves under WSL paths.
	gitdir = filepath.ToSlash(gitdir)
	switch {
	case strings.Contains(gitdir, "/.git/worktrees/") || strings.Contains(gitdir, "/worktrees/"):
		return linkedWorktree
	case strings.Contains(gitdir, "/.git/modules/") || strings.Contains(gitdir, "/modules/"):
		return submodule
	default:
		return notGit
	}
}
