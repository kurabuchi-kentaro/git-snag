# 0004. Sort modes toggle between tree and flat rendering

Date: 2026-07-30

## Context

The primary way users find deletion candidates is ordering — "oldest
first", "merged first" — not text search. But imposing such orders on the
nested branch tree of ADR 0003 visibly breaks it: siblings jump across
group boundaries and the connectors stop meaning anything. Prototyping
confirmed it reads as broken rather than sorted.

## Decision

Sorting and hierarchy are mutually exclusive per repository view:

- **TreeView** (default): the nested hierarchy, alphabetical per level,
  protected-containing subtrees pinned first (a group inherits the priority
  of its most prioritized descendant).
- **StaleFirst / FreshFirst / MergedFirst**: the hierarchy is abandoned and
  worktrees render as a flat list under the repository header, with full
  branch names (no ancestor rows to imply the prefix) and protected
  worktrees pinned first. MergedFirst tie-breaks by staleness.

The `s` key cycles all four; only repository-header grouping survives in
flat modes.

## Consequences

- Each mode is honest about what it shows: structure or order, never a
  mangled hybrid.
- The UI needs two build paths (tree.Build / tree.BuildFlat), kept trivial
  by sharing the leaf representation.
- Group collapse state is meaningless in flat modes and is simply unused
  there, not reset — switching back to TreeView restores it.
