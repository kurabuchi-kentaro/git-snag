// Package testutil builds real throwaway git repositories for integration
// tests. It shells out to git directly (never via internal/gitcli) so that
// gitcli's own tests are not tautological.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Repo is a real git repository rooted in a temporary directory that is
// cleaned up when the test finishes.
type Repo struct {
	T   testing.TB
	Dir string
}

// NewRepo creates a repository with an initial empty commit on `main`.
func NewRepo(t testing.TB) *Repo {
	t.Helper()
	return NewRepoAt(t, t.TempDir())
}

// NewRepoAt creates a repository at the given path (created if necessary)
// with an initial empty commit on `main`.
func NewRepoAt(t testing.TB, dir string) *Repo {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	r := &Repo{T: t, Dir: dir}
	r.Git("init", "-q", "-b", "main")
	r.Commit("initial commit")
	return r
}

// Git runs git with the given arguments in the repository root and fails the
// test on error.
func (r *Repo) Git(args ...string) string {
	r.T.Helper()
	return r.GitIn(r.Dir, args...)
}

// GitIn runs git in an arbitrary directory (e.g. a linked worktree) and
// fails the test on error.
func (r *Repo) GitIn(dir string, args ...string) string {
	r.T.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.T.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
	return string(out)
}

// Commit records an empty commit in the main worktree.
func (r *Repo) Commit(msg string) {
	r.T.Helper()
	r.Git("commit", "-q", "--allow-empty", "-m", msg)
}

// CommitIn records an empty commit inside the given worktree directory.
func (r *Repo) CommitIn(dir, msg string) {
	r.T.Helper()
	r.GitIn(dir, "commit", "-q", "--allow-empty", "-m", msg)
}

// WriteFile writes a file relative to the given directory.
func (r *Repo) WriteFile(dir, rel, content string) string {
	r.T.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		r.T.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.T.Fatalf("write %s: %v", path, err)
	}
	return path
}

// AddWorktree creates a linked worktree on a new branch and returns its path.
func (r *Repo) AddWorktree(branch string) string {
	r.T.Helper()
	path := filepath.Join(r.T.TempDir(), strings.ReplaceAll(branch, "/", "-"))
	r.Git("worktree", "add", "-q", "-b", branch, path)
	return path
}

// AddDetachedWorktree creates a linked worktree with a detached HEAD.
func (r *Repo) AddDetachedWorktree() string {
	r.T.Helper()
	path := filepath.Join(r.T.TempDir(), "detached")
	r.Git("worktree", "add", "-q", "--detach", path)
	return path
}

// SetupOrigin creates a bare repository, wires it up as `origin`, pushes
// main with an upstream, and points origin/HEAD at main.
func (r *Repo) SetupOrigin() string {
	r.T.Helper()
	bare := filepath.Join(r.T.TempDir(), "origin.git")
	r.GitIn(r.Dir, "init", "-q", "--bare", bare)
	r.Git("remote", "add", "origin", bare)
	r.Git("push", "-q", "-u", "origin", "main")
	r.Git("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	return bare
}
