# 0009. Default-branch detection and the no-origin fallback

Date: 2026-07-30

## Context

The "merged" tag compares each branch against the repository's default
branch. Repositories usually expose it as the `origin/HEAD` symbolic ref,
but local-only repositories have no origin at all, and the tool must not
fail on them.

## Decision

`gitcli.DefaultBranch` resolves `git symbolic-ref refs/remotes/origin/HEAD`
and strips the remote prefix. Any failure to resolve it is reported as an
error matching the sentinel `ErrNoDefaultBranch`; `enrich` treats that as
"skip merge detection for this repository" — every worktree simply carries
no `merged` tag, and nothing errors out.

## Consequences

- Local-only repositories work, at the cost of losing the merged signal
  there. Falling back to the main worktree's HEAD (so merge detection also
  works without a remote) is a possible future improvement, deliberately not
  done in v0.1 to keep the semantics of "merged" unambiguous: merged into
  what the *remote* considers the default branch.
- Repositories where `origin/HEAD` is unset (it can be created with
  `git remote set-head origin -a`) behave like no-origin repositories.
  This is acceptable for v0.1 and easy to revisit behind the same sentinel.
