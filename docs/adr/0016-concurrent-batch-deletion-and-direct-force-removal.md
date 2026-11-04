# 0016. Concurrent batch deletion, and `--force` removal from the start

Date: 2026-11-04

## Context

Deleting around 100 worktrees in one batch took minutes. `action.Execute`
had two costs that grew with the batch:

1. **Every removal ran twice when dirty, and paid a status scan when clean.**
   Following ADR 0005, `removeWorktree` first ran a plain
   `git worktree remove` and retried with `--force` on any failure. A plain
   removal runs a full status scan of the worktree before deleting it; on
   worktrees with a few thousand tracked files that scan dominated the
   per-item cost (about 1.06 s per item plain vs 0.16 s with `--force` in a
   local measurement).
2. **Items ran strictly one after another**, so the batch time was the sum
   of every removal.

Because the retry fired on *any* plain-removal failure, the final outcome of
the old sequence was always the outcome of `--force`; the plain attempt only
added time.

Not every git operation in a batch is safe to overlap within one repository:

- `git worktree prune` deletes the administrative entry of any worktree
  whose directory is missing — including one that a concurrent
  `worktree remove` has deleted the directory of but not yet unregistered.
- `git branch -d/-D` rewrites shared ref storage (`packed-refs`) under a
  lock, so concurrent deletions in one repository contend and can fail.

Concurrent `worktree remove` calls on different worktrees of the same
repository touch disjoint `$GIT_DIR/worktrees/<id>` entries and overlap
safely.

## Decision

**Remove worktrees with `--force` directly.** The plain attempt is dropped.
The consent model of ADR 0005 is unchanged — risk states are warned about on
the confirmation screen and one confirmation covers the batch — only the
escalation step it describes for worktree removal is gone. Locked worktrees
are still unlocked first; branch deletion still tries `-d` before `-D` so the
summary keeps distinguishing clean from forced deletions.

**Run the batch concurrently, in per-repository phases.** Repositories are
processed concurrently. Within a repository:

1. worktree removals run concurrently;
2. after all of them finish, prune runs (once per repository, as before);
3. then branch deletions (including branch-only items) run one at a time.

A single semaphore caps the git processes running across the whole batch at
**4**. Removal time is mostly unlinking files, and past a few concurrent
removals filesystem contention outweighs the overlap. Measured on ext4 under
WSL2, per-item removal time was 103 ms serial, 64 ms at 2, 37 ms at 4, and
62 ms at 12.

Results are still returned in item order, and one item's failure still never
aborts the rest.

## Consequences

- A 100-item batch (3000 tracked files and 5000 ignored files per worktree)
  went from 2 min 13 s to 26 s in a local measurement. The gain on other
  filesystems — notably `/mnt/c` under WSL — is unmeasured, and the
  remaining time is mostly the file deletion itself.
- Within a repository, a branch is no longer deleted right after its own
  worktree but after all of that repository's removals. If the context is
  cancelled in between, a worktree can be reported removed with its branch
  step not attempted; the UI runs batches with a background context, so this
  does not occur today.
- The cap of 4 is a constant tuned on one machine. Revisit it if users on
  other filesystems report slow batches.
- Removing large trees could be made near-instant by renaming the worktree
  directory aside, pruning, and deleting the moved tree later. That was not
  adopted: the deletion would have to outlive the TUI process.
