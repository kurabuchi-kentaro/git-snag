# 0006. Visual-mode range selection instead of press-and-hold

Date: 2026-07-30

## Context

Selecting a run of adjacent worktrees one space-press at a time is tedious.
The intuitive idea — hold space and move — is not reliably implementable in
a terminal: standard terminal I/O delivers key *repeats*, not press/release
pairs, so "how long was the key held" and "when was it released" are
unknowable outside enhanced-keyboard protocols (Kitty et al.) that cannot be
assumed across terminals.

## Decision

Adopt vim-style Visual mode, the established TUI substitute (vim, ranger):
`v` anchors at the focused row; `j`/`k` extend a live range whose selectable
leaves are *added* to whatever was selected before entering the mode;
`v` confirms; `Esc` reverts to the exact pre-visual selection; a mouse click
confirms and exits. While active, all other input — `/`, `m`, `s`, `d`,
even `q` — is ignored so the mode cannot be left in an ambiguous state.
Rows inside the range that are protected leaves are skipped, and group or
header rows never bulk-select their descendants from a range pass-through.

## Consequences

- Works on every terminal; no timing heuristics, no key-repeat tuning.
- Additive-with-revert semantics make the mode safe to experiment with:
  Esc always restores the previous state exactly.
- One more modal state in the UI model, covered by dedicated Update tests
  (`[v,j,j,v]` and `[v,j,j,Esc]` sequences).
