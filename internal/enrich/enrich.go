// Package enrich fills domain.Repo with per-worktree status: dirty, merged,
// unpushed, last-commit time, and protection flags.
package enrich

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
	"github.com/kurabuchi-kentaro/git-snag/internal/gitcli"
)

// maxConcurrentWorktrees bounds the per-repository worker pool: subprocess
// spawns dominate the cost, and unbounded fan-out gains nothing.
const maxConcurrentWorktrees = 8

// Enricher tags repositories with worktree status. cwd (captured once at
// startup) decides which worktree is "current" and therefore protected.
type Enricher struct {
	git *gitcli.Client
	cwd string
}

// New returns an Enricher that considers cwd the user's current location.
// Symlinks in cwd are resolved so containment checks match git-reported
// paths (macOS reports /private/var for /var, for example).
func New(cwd string) *Enricher {
	if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
		cwd = resolved
	}
	return &Enricher{git: gitcli.New(), cwd: cwd}
}

// Enrich lists the repository's worktrees and fills their status fields.
// Repo-wide facts (default branch, merged set) are resolved once per
// repository; per-worktree facts run in a bounded worker pool. A worktree
// that disappears mid-flight degrades to unenriched fields instead of
// aborting the others.
func (e *Enricher) Enrich(ctx context.Context, repoPath string) (domain.Repo, error) {
	repo := domain.Repo{Path: repoPath, Name: filepath.Base(repoPath)}

	worktrees, err := e.git.ListWorktrees(ctx, repoPath)
	if err != nil {
		return domain.Repo{}, err
	}

	merged := map[string]bool{}
	defaultBranch, err := e.git.DefaultBranch(ctx, repoPath)
	switch {
	case err == nil:
		repo.DefaultBranch = defaultBranch
		if m, mergedErr := e.git.MergedBranches(ctx, repoPath, defaultBranch); mergedErr == nil {
			merged = m
		}
		// Merges usually land on the remote (PRs), so the remote-tracking
		// default is often ahead of an un-pulled local one. Union both
		// targets so those merges still count; a missing origin/<default>
		// ref just skips this half.
		if m, mergedErr := e.git.MergedBranches(ctx, repoPath, "origin/"+defaultBranch); mergedErr == nil {
			for branch := range m {
				merged[branch] = true
			}
		}
	case errors.Is(err, gitcli.ErrNoDefaultBranch):
		// No origin: merge detection is skipped by contract (ADR 0009).
	default:
		return domain.Repo{}, err
	}

	branches, err := e.git.Branches(ctx, repoPath)
	if err != nil {
		return domain.Repo{}, err
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentWorktrees)
	for i := range worktrees {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			e.enrichWorktree(ctx, &worktrees[i], merged)
		}()
	}
	wg.Wait()

	repo.Worktrees = worktrees
	// Branch protection derives from the worktrees, so it must wait for
	// the pool above to settle IsCurrent.
	protectedWt := map[string]bool{}
	for _, w := range worktrees {
		if w.Protected() {
			protectedWt[w.Path] = true
		}
	}
	for i := range branches {
		b := &branches[i]
		b.Merged = merged[b.Name]
		b.Protected = b.Name == repo.DefaultBranch || protectedWt[b.WorktreePath]
	}
	repo.Branches = branches
	return repo, nil
}

// enrichWorktree fills the status of a single worktree in place. Failures on
// individual probes (e.g. the directory vanished) leave that field at its
// zero value rather than propagating.
func (e *Enricher) enrichWorktree(ctx context.Context, wt *domain.Worktree, merged map[string]bool) {
	if wt.Branch != "" {
		wt.Merged = merged[wt.Branch]
	}
	wt.IsCurrent = pathContains(wt.Path, e.cwd)
	if wt.Prunable {
		// The directory is gone; working-tree probes are meaningless.
		return
	}
	if dirty, err := e.git.IsDirty(ctx, wt.Path); err == nil {
		wt.Dirty = dirty
	}
	if count, hasUpstream, err := e.git.UnpushedCount(ctx, wt.Path); err == nil {
		wt.UnpushedCount = count
		wt.HasUpstream = hasUpstream
	}
	if ts, err := e.git.LastCommitTime(ctx, wt.Path); err == nil {
		wt.LastCommitTime = ts
	}
}

// pathContains reports whether candidate is dir itself or nested below it.
func pathContains(dir, candidate string) bool {
	rel, err := filepath.Rel(dir, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}
