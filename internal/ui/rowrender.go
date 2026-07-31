package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// relTimeCap is the ceiling of relative timestamps: precision past six
// months carries no decision value when hunting stale worktrees.
const relTimeCap = "6mo+ ago"

// timeColWidth is the fixed timestamp column, sized to the widest value
// (relTimeCap) so the tag cluster aligns across rows.
const timeColWidth = len(relTimeCap)

// RelativeTime renders a compact relative timestamp: minutes under an hour,
// hours under a day, days up to 30, months beyond that, capped at relTimeCap.
func RelativeTime(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	switch {
	case days <= 30:
		return fmt.Sprintf("%dd ago", days)
	case days < 180:
		return fmt.Sprintf("%dmo ago", days/30)
	default:
		return relTimeCap
	}
}

// connectorPrefix renders the box-drawing prefix for a row from its
// sibling-position ancestry (tree.Row.IsLast semantics).
func connectorPrefix(isLast []bool) string {
	var b strings.Builder
	for _, last := range isLast[:len(isLast)-1] {
		if last {
			b.WriteString("    ")
		} else {
			b.WriteString("│   ")
		}
	}
	if isLast[len(isLast)-1] {
		b.WriteString("└── ")
	} else {
		b.WriteString("├── ")
	}
	return b.String()
}

// pathConnectorPrefix renders the guide continuation under a leaf's
// connector so the tree's vertical lines run unbroken through path lines:
// ancestor bars, then a bar when the leaf has later siblings.
func pathConnectorPrefix(isLast []bool) string {
	var b strings.Builder
	for _, last := range isLast[:len(isLast)-1] {
		if last {
			b.WriteString("    ")
		} else {
			b.WriteString("│   ")
		}
	}
	if isLast[len(isLast)-1] {
		b.WriteString("    ")
	} else {
		b.WriteString("│   ")
	}
	return b.String()
}

// rowState carries the per-row display states, which may all hold at once.
type rowState struct {
	focused  bool
	selected bool
	inRange  bool
	// muted dims the name: the repository's default branch reads as
	// infrastructure, not a deletion candidate.
	muted bool
}

// gutter renders the one-cell left margin every row reserves: a purple bar
// inside the visual-mode range, a space otherwise.
func gutter(st rowState) string {
	if st.inRange {
		return bgIf(styleAccent, st.focused).Render(gutterBar)
	}
	return bgIf(lipgloss.NewStyle(), st.focused).Render(" ")
}

// bgIf paints the focused-row background onto a segment style; every
// segment of a focused row must carry it, or inner style resets would
// punch holes in the highlight.
func bgIf(s lipgloss.Style, focused bool) lipgloss.Style {
	if focused {
		return s.Background(colorFocus)
	}
	return s
}

// tagPart is one status tag: a glyph (plus optional label or count) and
// its semantic color.
type tagPart struct {
	text  string
	style lipgloss.Style
}

// tagParts returns the status tag cluster for a worktree.
func tagParts(w domain.Worktree, ic iconSet) []tagPart {
	var tags []tagPart
	if w.IsCurrent {
		tags = append(tags, tagPart{ic.current, styleDim})
	}
	if w.IsMain {
		tags = append(tags, tagPart{ic.main, styleDim})
	}
	if w.Dirty {
		tags = append(tags, tagPart{ic.dirty + " dirty", styleWarn})
	}
	if w.Locked {
		tags = append(tags, tagPart{ic.locked + " locked", styleInfo})
	}
	if w.Merged {
		tags = append(tags, tagPart{ic.merged + " merged", styleGood})
	}
	if w.UnpushedCount > 0 {
		tags = append(tags, tagPart{fmt.Sprintf("%s%d", ic.unpushed, w.UnpushedCount), styleSync})
	}
	if w.Prunable {
		tags = append(tags, tagPart{ic.prunable + " gone", styleBad})
	}
	return tags
}

// truncate shortens plain (unstyled) text to the given display width,
// appending an ellipsis.
func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// renderLeafLine renders a worktree's first display line: gutter, connector
// prefix, selection mark, name, then tags and relative time flush right.
func renderLeafLine(prefix string, w domain.Worktree, name string, width int, now time.Time, st rowState, ic iconSet) string {
	width-- // the gutter owns the first cell

	// No placeholder mark: the check appears only on selected rows, and
	// protection reads from the dimmed name instead of a dash.
	mark := ""
	if st.selected {
		mark = "✓ "
	}

	// The timestamp gets a fixed right-aligned column so tags end at the
	// same cell on every row regardless of how long "ago" is.
	timeStr := fmt.Sprintf("%*s", timeColWidth, RelativeTime(w.LastCommitTime, now))
	tags := tagParts(w, ic)
	rightWidth := timeColWidth
	for _, t := range tags {
		rightWidth += lipgloss.Width(t.text) + 1 // trailing gap before next part
	}
	if len(tags) > 0 {
		rightWidth++ // two-cell gap between tags and the time column
	}

	nameStyle := styleBranch
	switch {
	case st.selected:
		nameStyle = styleSelected
	case w.Protected() || st.muted:
		nameStyle = styleProtected
	}

	fixed := lipgloss.Width(prefix) + lipgloss.Width(mark)
	nameAvail := width - fixed - rightWidth - 1
	if nameAvail < 1 {
		// Too narrow for the full layout: name and timestamp only.
		line := truncate(mark+name, max(width-lipgloss.Width(timeStr)-1, 1))
		return gutter(st) + bgIf(styleDim, st.focused).Render(padLine(line, max(width-lipgloss.Width(timeStr), 1))+timeStr)
	}
	if lipgloss.Width(name) > nameAvail {
		name = truncate(name, nameAvail)
	}
	pad := width - fixed - lipgloss.Width(name) - rightWidth
	if pad < 1 {
		pad = 1
	}

	var b strings.Builder
	b.WriteString(gutter(st))
	b.WriteString(bgIf(styleDim, st.focused).Render(prefix))
	if mark != "" {
		b.WriteString(bgIf(styleSelected, st.focused).Render(mark))
	}
	b.WriteString(bgIf(nameStyle, st.focused).Render(name))
	b.WriteString(bgIf(lipgloss.NewStyle(), st.focused).Render(strings.Repeat(" ", pad)))
	for _, t := range tags {
		b.WriteString(bgIf(t.style, st.focused).Render(t.text))
		b.WriteString(bgIf(lipgloss.NewStyle(), st.focused).Render(" "))
	}
	if len(tags) > 0 {
		b.WriteString(bgIf(lipgloss.NewStyle(), st.focused).Render(" "))
	}
	b.WriteString(bgIf(styleDim, st.focused).Render(timeStr))
	return b.String()
}

// renderPathLine renders a worktree's second display line: guide
// continuation, then the filesystem path, annotated when the directory no
// longer exists.
func renderPathLine(prefix string, w domain.Worktree, width int, st rowState) string {
	width-- // the gutter owns the first cell
	path := w.Path
	if w.Prunable {
		path += " (missing)"
	}
	line := truncate(prefix+path, width)
	pad := width - lipgloss.Width(line)
	if pad > 0 && st.focused {
		line += strings.Repeat(" ", pad)
	}
	return gutter(st) + bgIf(styleDim, st.focused).Render(line)
}
