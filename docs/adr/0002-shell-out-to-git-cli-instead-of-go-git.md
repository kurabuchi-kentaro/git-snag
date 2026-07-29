# 0002. Shell out to the git CLI instead of using go-git

Date: 2026-07-30

## Context

git-snag needs worktree listing/removal, branch deletion, merge detection,
dirty checks, and upstream comparisons. Two implementation routes exist: the
pure-Go go-git library, or spawning the user's local `git` binary via
`os/exec`.

go-git's worktree support is incomplete (no `worktree list/remove/lock`
equivalents), and any reimplementation risks subtle behavioral drift from
real git — exactly the kind of drift that is dangerous in a tool whose job
is deleting things.

## Decision

All git operations shell out to the local `git` binary through
`internal/gitcli`. Machine-readable interfaces (`--porcelain`, `--format`)
are used wherever git offers them. The parser ignores unknown porcelain
attributes for forward compatibility, and failures carry git's stderr, with
`ErrNotRepository` as the only sentinel callers branch on (plus
`ErrNoDefaultBranch`, see ADR 0009).

## Consequences

- Behavior always matches the user's own git version, config, and hooks.
- git becomes a runtime dependency; startup verifies it is on PATH.
- Every operation pays subprocess overhead, mitigated by batching
  (one `branch --merged` per repo) and a bounded worker pool in enrich.
- Tests run against real repositories instead of mocks, which is what the
  testing policy wants anyway.
