# 0007. The deletion explosion animation, adapted from lazygit

Date: 2026-07-30

## Context

Deleting worktrees is the tool's one destructive moment; a short celebratory
animation gives it weight and personality. lazygit's "Nuke working tree"
effect (`pkg/gui/controllers/workspace_reset_controller.go`, `Explode` /
`getExplodeImage`, MIT licensed) is the reference: a radial burst of random
symbols with a square-root progress easing, a trailing inner shockwave, a
2× vertical distance scale to compensate for the terminal cell aspect
ratio, and ~25 frames at 20 ms.

## Decision

Port the frame-generation math faithfully but re-express the driver in
Bubble Tea idiom: lazygit renders from a sleeping goroutine with UI-thread
callbacks (gocui architecture); git-snag advances frames via a `tea.Tick`
message loop so the animation stays inside the ordinary Update cycle and is
unit-testable by injecting tick messages. Intensity scales with the batch
size (1 / 2–4 / 5+, the last tier adding a pane shake). The animation runs
concurrently with the actual deletion; the summary appears when both are
done. Any key skips it instantly, and it can be disabled outright.

Credit lazygit explicitly in three places: this ADR, the README, and the
code comment at the top of `internal/ui/explosion.go`. The name "git-snag"
deliberately carries no "lazy" branding — the homage is this precise
attribution, not an implied family membership.

## Consequences

- The effect stays skippable, disableable, and short (≤ ~700 ms even at the
  top tier), so it never gates the actual work.
- Frame rendering is a pure, seeded function — deterministic in tests.
- The tick-driven port is behaviorally equivalent but not line-for-line
  identical to lazygit's; drift from upstream improvements is possible and
  acceptable.
