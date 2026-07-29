# 0010. Protected worktrees are never selectable

Date: 2026-07-30

## Context

git itself refuses `git worktree remove` for the main worktree and for the
worktree the command runs inside. A deletion tool that lets users select
either would only fail late, at execution time, with a confusing git error.

## Decision

Two worktrees per repository are *protected*: the main worktree (first entry
of `worktree list`) and the worktree containing the process's working
directory at launch (`os.Getwd()`, matched by path containment so a nested
cwd still counts). Protected worktrees are shown in the tree — greyed, with
their own tags — but excluded from every selection mechanism: direct toggle,
group/repository recursive select, and Visual-mode ranges.

## Consequences

- Deletion can never fail on git's own refusal rules; those cases are
  unreachable by construction.
- "Current" is decided once at startup. A user who cd's elsewhere in another
  terminal mid-session keeps the launch-time protection, which is the
  conservative direction.
- The tree still shows the whole picture: users can see the protected
  worktrees exist and why they are not selectable.
