# 0014. Branch mode for worktree-less branch cleanup

Date: 2026-08-01

Amended 2026-08-01: the default-branch root anchor described below was
dropped after review. Nesting every branch under the default branch reads
as an ancestry claim that branch names cannot make, and it left the
selection-propagation rule of the anchor row unreadable (does space on
main select everything beneath it?). The default branch now renders as a
muted sibling pinned first — the same shape as the worktree view. The same
review added single-child namespace compaction (VS Code's compact-folders
convention) to both trees: a namespace that disambiguates nothing folds
into its child's label, e.g. `feature/uc1-backend` as one row.

## Context

git-snag only shows branches that have a worktree. In real repositories
most stale refs are the opposite: local branches whose worktree never
existed or is long gone. Cleaning those still means `git branch -d` by
hand, one repository at a time — exactly the chore this tool exists to
remove.

A branch view raises questions a worktree view never had to answer:

- Branches have no recorded ancestry. "Which branch did this fork from"
  is a merge-base heuristic — ambiguous after merges, unstable across
  rebases, and O(n²) subprocess calls to compute.
- A branch checked out in a worktree cannot be deleted until that
  worktree is removed; the main and current worktrees cannot be removed
  at all.
- `git branch -d` refuses unmerged branches.
- Squash-merged branches are never ancestors of the default branch, so
  `--merged` misses them; the one reliable signal is a configured
  upstream that no longer exists (`[gone]`).

## Decision

`b` toggles a **branch mode** view; the same key returns. It is a
persistent view mode like the sort modes, not a transient gesture: `q`
still quits the app and `esc` still clears the selection. The footer
carries a blue ` BRANCHES ` badge (blue = which view; the purple
` VISUAL ` badge = what gesture; both show side by side, persistent
first). The status line counts read `N repos · M branches` and turn
into the same fractions while selecting.

**Shape.** Each repository shows its local branches nested under the
default branch as the visual root, using the existing slash-delimited
name hierarchy (ADR 0003) — not commit ancestry, for the reasons above.
Branches checked out in a worktree carry a blue `+` before the name
(`git branch`'s own convention). All existing view operations work
unchanged: filter, merged-only, flat sort modes (repo-spanning), info
levels, collapse, visual mode.

**Data.** Branches are read eagerly during enrichment: one
`git for-each-ref refs/heads` per repository supplies name, head,
committer date, upstream ahead/gone state, and the checkout path. The
merged set is shared with the worktree view. Branch rows show merged,
`↑n`, a dedicated *gone* tag for deleted upstreams, and the relative
time; `+` rows add their worktree's dirty/locked tags, since deleting
them removes that worktree.

**Deletion.** Selection is keyed by repo path + branch name and cleared
on every mode switch, so nothing invisible is ever pending. A bare
branch deletes with `-d`, escalating to `-D` on refusal — the same
silent-escalation contract as locked worktrees (ADR 0005), with an
`unmerged (will force-delete)` note in the confirm modal and a forced
marker in the summary. A `+` branch goes through the existing
worktree-plus-branch plan unchanged. The default branch and branches
checked out in protected worktrees (ADR 0010) are shown muted and are
never selectable.

**Merged-only in branch mode** matches on merged *or* upstream-gone:
in squash-merge workflows the gone state is the working definition of
"merged", and the filter exists to line up exactly the safe deletions.

## Consequences

- Worktree-less branches become first-class cleanup targets across many
  repositories at once, with the same selection grammar as worktrees.
- The name-based tree means stacked branches do not nest by ancestry;
  the tree answers "what is there", not "what forked from what".
- Eager loading costs one subprocess per repository at startup and
  keeps the UI layer free of git calls (the layer rule stands).
- Deleting a `+` branch deletes a working directory. The dirty tag on
  the row and the confirm modal both surface this before it happens.
- Mode switches drop the pending selection by design; users who select
  in one mode must delete in that mode.
