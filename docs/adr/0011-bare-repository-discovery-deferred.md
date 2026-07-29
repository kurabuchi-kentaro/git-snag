# 0011. Bare-repository discovery is deferred (v0.1 limitation)

Date: 2026-07-30

## Context

Some worktree power-users keep no regular checkout at all: they `git clone
--bare` and attach every working tree via `git worktree add`. The scanner
discovers repositories structurally, by the presence of a `.git` entry — but
a bare repository's directory *is* the gitdir and contains no `.git` child,
so this layout is invisible to the walk. Detecting it would require a
costlier heuristic (probing directories for HEAD/objects/refs, or running
`git rev-parse --is-bare-repository` speculatively).

## Decision

v0.1 does not discover bare repositories. The walk classifies only:
`.git` directory → repository; `.git` file → linked worktree or submodule
(via the gitdir path), both skipped. Repositories discovered this way are
not descended into.

## Consequences

- Bare + worktrees setups see none of their worktrees in git-snag v0.1.
  This is a known, documented limitation rather than a bug.
- If issue reports show real demand, discovery can add a bounded heuristic
  (e.g. treat `*.git` directories containing HEAD as bare repositories)
  without changing any other layer, since the scanner only emits paths.
