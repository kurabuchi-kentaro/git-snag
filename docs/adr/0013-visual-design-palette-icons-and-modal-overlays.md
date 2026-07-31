# 13. Visual design: ANSI palette, Nerd Font icons, and modal overlays

Date: 2026-07-31

## Status

Accepted

## Context

The first working TUI used four bare lipgloss styles (bold, faint, reverse)
and emoji status tags. Feedback from real use called it too plain, and
surfaced concrete defects:

- Tree guide lines break at every worktree path line, because the path row
  indents with spaces instead of continuing the `│` guides — and the indent
  width was computed with `len()` over multi-byte box-drawing runes, pushing
  paths deeper than intended.
- Emoji tags cannot be tinted by ANSI colors, render with inconsistent
  widths across terminals (notably `⬆️` with its variation selector), and
  waste the `UnpushedCount` data the enrich layer already provides.
- Full-screen phase switches (browse → confirm → explosion → summary →
  browse) discard the tree context the user was just looking at.
- Reverse-video focus inverts tag colors and reads as harsh.

## Decision

### Palette: ANSI base plus one fixed accent

State colors use the terminal's ANSI palette so they follow the user's
theme: merged = green, dirty = yellow, locked = blue, prunable = red,
unpushed = cyan. Green is reserved for state/success; the single fixed
accent is purple (truecolor, degrading to ANSI magenta) and marks
*operations*: selection, visual-range, filter prompt, modal frames.
Relative times, paths, tree guides, selection dots, counts, and protected
markers are faint. Color-profile degradation and `NO_COLOR` are delegated
to bubbletea/lipgloss v2 detection.

### Text hierarchy: ls convention

Directory group rows (`feat/`) render blue and bold, following the
`ls`/`eza` convention; branch names stay in the default foreground as the
primary content; repository names are bold. The clash with locked-blue is
accepted: locked appears only in the right-hand tag cluster.

### Icons: Nerd Fonts by default, plain Unicode as fallback

Status tags are single-width glyphs so ANSI colors apply. The default set
uses Nerd Font code points; `icons = "unicode"` in the config (or
`--icons unicode`) switches to plain Unicode symbols for terminals without
a patched font. Unpushed renders with its count (`↑3`). Emoji are removed.

### Row states

Focus paints a subtle background over the whole row (reverse video is
dropped). Selection turns the mark and name purple. Visual-mode range shows
a purple `▌` bar in the leftmost column. The three states compose.

### Modal overlays instead of full-screen switches

Confirm, summary, and help render as centered modals over the (dimmed)
tree, composited with lipgloss v2 Canvas/Layer; the backdrop dim sets
`uv.AttrFaint` on background cells. The delete flow is three acts in one
frame: the confirm list, then the explosion animation *inside the same
frame* (lazygit plays its nuke inside the Files panel; a full-screen blast
was judged too loud), then the summary. The frame's size is fixed when the
confirm opens so the acts don't jump. Modals cap at
`min(width-8, 72) × min(height-4, content)`; longer lists scroll internally
following the `j`/`k` cursor with an `n/N` position indicator. The frame is
always purple; danger is conveyed inside the confirm (red title, yellow
warnings), and the summary marks results `✓` green / `✗` red / `–` faint.

### Footer: emphasized keys, contextual hints

Keys render bold, descriptions faint. The hint line swaps with context:
browsing, selection pending (`N selected · d delete …` with purple
emphasis), and visual mode (purple reverse `VISUAL` badge).

## Consequences

- `internal/ui` gains a styles vocabulary and two icon tables; tests and
  the README legend switch from emoji to the new glyphs.
- The config schema grows a `ui.icons` key (ADR 0008's global-only rule
  unchanged); `--icons` wins over the file.
- Tree guides continue through path lines; the byte-length indent bug is
  fixed by rendering guides instead of spaces.
- Rendering paths that previously returned plain strings now return styled
  strings; row-level tests assert on content substrings, which survive
  styling, but width-sensitive tests must measure with `lipgloss.Width`.
- The explosion no longer owns the screen; its canvas is the modal
  interior, and tier-based shake jitters the layer offset instead of the
  whole frame.
