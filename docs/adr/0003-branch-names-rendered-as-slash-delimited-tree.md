# 0003. Branch names render as a slash-delimited tree

Date: 2026-07-30

## Context

Worktree-heavy repositories conventionally namespace branches with slashes
(`feature/rate-limit`, `fix/retry-logic`). A flat list repeats those
prefixes on every row and hides the natural grouping. Users hunting for
deletable worktrees think in terms of these prefix families — including
spotting naming accidents like `feature/fix/login-redirect`.

## Decision

In the default view, each repository's worktrees are grouped by splitting
branch names on `/`, rendered with `tree`-command box-drawing connectors
(`├──`, `└──`, `│`). Group nodes are purely synthetic — git's ref
namespacing guarantees a branch `feature` and a branch `feature/x` cannot
coexist, so a group can never collide with a real worktree. Leaves show only
their last segment (ancestors imply the prefix). Detached-HEAD worktrees
have no branch and appear at the repository root with a short-SHA label.
Groups collapse and select as units.

## Consequences

- The hierarchy mirrors how users actually name and mentally organize
  branches; prefix families select and collapse in one gesture.
- Arbitrary re-sorting cannot be expressed inside the hierarchy — that
  tension is resolved by ADR 0004.
- Rendering needs sibling-position tracking (see tree.Flatten's Row.IsLast)
  rather than simple indentation.
