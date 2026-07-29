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

// tagsFor returns the status icon cluster for a worktree.
func tagsFor(w domain.Worktree) string {
	var tags []string
	if w.IsCurrent {
		tags = append(tags, iconCurrent)
	}
	if w.IsMain {
		tags = append(tags, iconMain)
	}
	if w.Dirty {
		tags = append(tags, iconDirty)
	}
	if w.Locked {
		tags = append(tags, iconLocked)
	}
	if w.Merged {
		tags = append(tags, iconMerged)
	}
	if w.UnpushedCount > 0 {
		tags = append(tags, iconUnpushed)
	}
	if w.Prunable {
		tags = append(tags, iconPrunable)
	}
	return strings.Join(tags, " ")
}

// truncate shortens s to the given display width, appending an ellipsis.
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

// padBetween joins left and right with spaces so the whole line occupies
// exactly width cells, truncating left when it does not fit.
func padBetween(left, right string, width int) string {
	rw := lipgloss.Width(right)
	avail := width - rw - 1
	if avail < 1 {
		return truncate(right, width)
	}
	if lipgloss.Width(left) > avail {
		left = truncate(left, avail)
	}
	pad := width - lipgloss.Width(left) - rw
	return left + strings.Repeat(" ", pad) + right
}

// renderLeafLine renders a worktree's first display line: connector prefix,
// selection mark, name, then tags and relative time flush right.
func renderLeafLine(prefix string, w domain.Worktree, name string, width int, now time.Time, focused, selected bool) string {
	mark := "·"
	switch {
	case w.Protected():
		mark = "–"
	case selected:
		mark = "✓"
	}
	left := prefix + mark + " " + name
	right := strings.TrimSpace(tagsFor(w) + "  " + RelativeTime(w.LastCommitTime, now))
	line := padBetween(left, right, width)
	if focused {
		return styleFocused.Render(line)
	}
	if w.Protected() {
		return styleProtected.Render(line)
	}
	return line
}

// renderPathLine renders a worktree's second display line: its filesystem
// path, annotated when the directory no longer exists.
func renderPathLine(prefix string, w domain.Worktree, width int) string {
	path := w.Path
	if w.Prunable {
		path += " (missing)"
	}
	return styleDim.Render(truncate(prefix+path, width))
}
