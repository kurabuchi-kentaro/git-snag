# 0005. Warn about dirty/locked/unpushed — then force, never block

Date: 2026-07-30

## Context

git refuses `worktree remove` on dirty worktrees without `--force` and on
locked worktrees without unlocking; branches with unmerged or unpushed
commits refuse `branch -d`. A cleanup tool has to decide whether these
states block deletion or merely inform it. Blocking would force users back
to the shell for exactly the cases the tool exists to handle; deleting
silently would destroy work without consent.

## Decision

Risk states never block and are never silent. The confirmation screen
annotates each item with its warnings — uncommitted changes, locked (will
force-unlock), unpushed commit count — and one explicit confirmation covers
the whole batch. Execution then escalates as needed: locked worktrees are
unlocked first, a refused plain removal is retried with `--force`, and a
refused `branch -d` falls back to `-D` with the escalation recorded in the
result so the summary can distinguish clean from forced deletions.

Batches are also resilient: one item's failure (e.g. filesystem permissions)
never aborts the rest, and the summary reports worktree and branch outcomes
separately per item (REQ-A8).

## Consequences

- The tool can actually complete its job on messy worktrees, with informed
  consent concentrated in one screen.
- A user who confirms carelessly can lose uncommitted work — the warnings
  and per-item exclusion checkboxes are the mitigation, by design.
- Force-escalation on *any* plain-removal failure means an unforeseen git
  refusal is also overridden after confirmation; acceptable because the
  scope of `worktree remove --force` is limited to that worktree.
