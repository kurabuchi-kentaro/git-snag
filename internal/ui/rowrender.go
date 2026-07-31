package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// relTimeCap is the ceiling of relative timestamps: precision past six
// months carries no decision value when hunting stale worktrees. The
// warning color marks the cap, so no "+" is needed and every value fits
// in three cells.
const relTimeCap = "6mo"

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
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	switch {
	case days <= 30:
		return fmt.Sprintf("%dd", days)
	case days < 180:
		return fmt.Sprintf("%dmo", days/30)
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
	// muted grays the whole entry: the repository's default branch reads
	// as infrastructure, not a deletion candidate.
	muted bool
}

// gutter renders the one-cell left margin every row reserves: a purple bar
// inside the visual-mode range, a space otherwise.
func gutter(st rowState) string {
	if st.inRange {
		return bgIf(styleAccent, st).Render(gutterBar)
	}
	return bgIf(lipgloss.NewStyle(), st).Render(" ")
}

// bgIf paints the row background onto a segment style — the theme gray for
// focus, the dark accent for selection (focus wins when both hold). Every
// segment of the row must carry it, or inner style resets would punch
// holes in the highlight.
func bgIf(s lipgloss.Style, st rowState) lipgloss.Style {
	switch {
	case st.focused:
		return s.Background(colorFocus)
	case st.selected:
		return s.Background(colorSelBg)
	}
	return s
}

// tagPart is one status tag: a glyph (plus optional label or count) and
// its semantic color.
type tagPart struct {
	text  string
	style lipgloss.Style
}

// worktreeTagParts emits the tags derived from a worktree's local state:
// current/main markers, then dirty and locked. Shared between the worktree
// view and branch mode's + rows.
func worktreeTagParts(w domain.Worktree, ic iconSet) []tagPart {
	var tags []tagPart
	if w.IsCurrent {
		tags = append(tags, tagPart{ic.current, styleDim})
	}
	if w.IsMain {
		tags = append(tags, tagPart{ic.main, styleDim})
	}
	if w.Dirty {
		tags = append(tags, tagPart{ic.dirty, styleWarn})
	}
	if w.Locked {
		tags = append(tags, tagPart{ic.locked, styleInfo})
	}
	return tags
}

// mutedTagParts is the tag cluster of a muted (protected) row: its status
// noise is suppressed — only the home marker remains.
func mutedTagParts(w *domain.Worktree, ic iconSet) []tagPart {
	if w != nil && w.IsMain {
		return []tagPart{{ic.main, styleDim}}
	}
	return nil
}

// tagParts returns the status tag cluster for a worktree: bare glyphs, with
// the text legend living in the help modal.
func tagParts(w domain.Worktree, ic iconSet, muted bool) []tagPart {
	if muted {
		return mutedTagParts(&w, ic)
	}
	tags := worktreeTagParts(w, ic)
	if w.Merged {
		tags = append(tags, tagPart{ic.merged, styleGood})
	}
	if w.UnpushedCount > 0 {
		tags = append(tags, tagPart{fmt.Sprintf("%s%d", ic.unpushed, w.UnpushedCount), styleSync})
	}
	if w.Prunable {
		tags = append(tags, tagPart{ic.prunable, styleBad})
	}
	return tags
}

// branchTagParts returns the tag cluster of a branch-mode row: worktree
// status first (deleting a + branch removes that worktree, so its dirty and
// locked states are decision inputs), then the branch's own remote state.
func branchTagParts(b domain.Branch, w *domain.Worktree, ic iconSet, muted bool) []tagPart {
	if muted {
		return mutedTagParts(w, ic)
	}
	var tags []tagPart
	if w != nil {
		tags = worktreeTagParts(*w, ic)
	}
	if b.Merged {
		tags = append(tags, tagPart{ic.merged, styleGood})
	}
	if b.UnpushedCount > 0 {
		tags = append(tags, tagPart{fmt.Sprintf("%s%d", ic.unpushed, b.UnpushedCount), styleSync})
	}
	if b.UpstreamGone {
		tags = append(tags, tagPart{ic.gone, styleGood})
	}
	return tags
}

// rightSide computes the reserved right side of a leaf line — the tag
// cluster plus the fixed time column — returning the formatted time, its
// style (the capped value carries the warning color: half a year of silence
// is what this tool hunts), and the total reserved width.
func rightSide(tags []tagPart, t time.Time, now time.Time) (timeStr string, timeStyle lipgloss.Style, rightWidth int) {
	rel := RelativeTime(t, now)
	timeStyle = styleDim
	if rel == relTimeCap {
		timeStyle = styleWarn
	}
	timeStr = fmt.Sprintf("%*s", timeColWidth, rel)
	rightWidth = timeColWidth
	for _, t := range tags {
		rightWidth += lipgloss.Width(t.text) + 1 // trailing gap before next part
	}
	if len(tags) > 0 {
		rightWidth++ // two-cell gap between tags and the time column
	}
	return timeStr, timeStyle, rightWidth
}

// writeTagsAndTime emits the reserved right side rightSide budgeted for.
func writeTagsAndTime(b *strings.Builder, tags []tagPart, timeStr string, timeStyle lipgloss.Style, st rowState) {
	for _, t := range tags {
		b.WriteString(bgIf(t.style, st).Render(t.text))
		b.WriteString(bgIf(lipgloss.NewStyle(), st).Render(" "))
	}
	if len(tags) > 0 {
		b.WriteString(bgIf(lipgloss.NewStyle(), st).Render(" "))
	}
	b.WriteString(bgIf(timeStyle, st).Render(timeStr))
}

// renderBranchLine renders a branch-mode leaf line: gutter, prefix, mark,
// a blue + for worktree-carrying branches (git branch's convention), the
// branch name in the branch color, then tags and time flush right. Same
// reserved-right-side budgeting as renderLeafLine.
func renderBranchLine(prefix string, br domain.Branch, w *domain.Worktree, name string, width int, now time.Time, st rowState, ic iconSet) string {
	width-- // the gutter owns the first cell

	mark := ""
	if st.selected {
		mark = "✓ "
	}
	plus := ""
	if br.HasWorktree() {
		plus = "+ "
	}

	tags := branchTagParts(br, w, ic, st.muted)
	timeStr, timeStyle, rightWidth := rightSide(tags, br.LastCommitTime, now)

	nameStyle := styleAccent
	if st.muted {
		nameStyle = styleDim
	}

	avail := width - rightWidth - 1 - lipgloss.Width(mark) - lipgloss.Width(plus)
	if lipgloss.Width(prefix) > avail-8 {
		prefix = truncate(prefix, max(avail-8, 1))
	}
	avail -= lipgloss.Width(prefix)
	if lipgloss.Width(name) > avail {
		name = truncate(name, max(avail, 1))
	}

	used := lipgloss.Width(prefix) + lipgloss.Width(mark) + lipgloss.Width(plus) + lipgloss.Width(name)
	pad := max(width-used-rightWidth, 1)

	var b strings.Builder
	b.WriteString(gutter(st))
	b.WriteString(bgIf(styleDim, st).Render(prefix))
	if mark != "" {
		b.WriteString(bgIf(lipgloss.NewStyle(), st).Render(mark))
	}
	if plus != "" {
		b.WriteString(bgIf(styleWorktree, st).Render(plus))
	}
	b.WriteString(bgIf(nameStyle, st).Render(name))
	b.WriteString(bgIf(lipgloss.NewStyle(), st).Render(strings.Repeat(" ", pad)))
	writeTagsAndTime(&b, tags, timeStr, timeStyle, st)
	return b.String()
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

// branchAnnotation renders the " (branch)" suffix a leaf carries after its
// directory name: the full branch name, or the detached-HEAD label.
func branchAnnotation(w domain.Worktree) string {
	if w.Detached() {
		sha := w.HeadSHA
		if len(sha) > 7 {
			sha = sha[:7]
		}
		return "(detached: " + sha + ")"
	}
	return "(" + w.Branch + ")"
}

// renderLeafLine renders a worktree's first display line: gutter, connector
// prefix, selection mark, directory name with its branch annotation, then
// tags and relative time flush right. The right side is reserved space —
// tags must never be pushed out, so the left side truncates to fit
// (annotation first, then the name).
func renderLeafLine(prefix string, w domain.Worktree, name string, width int, now time.Time, st rowState, ic iconSet) string {
	width-- // the gutter owns the first cell

	// No placeholder mark: the check appears only on selected rows.
	mark := ""
	if st.selected {
		mark = "✓ "
	}

	tags := tagParts(w, ic, st.muted)
	timeStr, timeStyle, rightWidth := rightSide(tags, w.LastCommitTime, now)

	ann := branchAnnotation(w)
	nameStyle, annStyle := styleWorktree, styleAccent
	if w.Detached() {
		annStyle = styleDim
	}
	if st.muted {
		nameStyle, annStyle = styleDim, styleDim
	}

	// Budget the left side: prefix, then name, then annotation, each
	// giving way to the reserved right side.
	avail := width - rightWidth - 1 - lipgloss.Width(mark)
	if lipgloss.Width(prefix) > avail-8 {
		prefix = truncate(prefix, max(avail-8, 1))
	}
	avail -= lipgloss.Width(prefix)
	if lipgloss.Width(name) > avail {
		name = truncate(name, max(avail, 1))
	}
	if annAvail := avail - lipgloss.Width(name) - 1; annAvail < 2 {
		ann = ""
	} else if lipgloss.Width(ann) > annAvail {
		ann = truncate(ann, annAvail)
	}

	used := lipgloss.Width(prefix) + lipgloss.Width(mark) + lipgloss.Width(name)
	if ann != "" {
		used += 1 + lipgloss.Width(ann)
	}
	pad := max(width-used-rightWidth, 1)

	var b strings.Builder
	b.WriteString(gutter(st))
	b.WriteString(bgIf(styleDim, st).Render(prefix))
	if mark != "" {
		b.WriteString(bgIf(lipgloss.NewStyle(), st).Render(mark))
	}
	b.WriteString(bgIf(nameStyle, st).Render(name))
	if ann != "" {
		b.WriteString(bgIf(lipgloss.NewStyle(), st).Render(" "))
		b.WriteString(bgIf(annStyle, st).Render(ann))
	}
	b.WriteString(bgIf(lipgloss.NewStyle(), st).Render(strings.Repeat(" ", pad)))
	writeTagsAndTime(&b, tags, timeStr, timeStyle, st)
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
	if pad > 0 && (st.focused || st.selected) {
		line += strings.Repeat(" ", pad)
	}
	return gutter(st) + bgIf(styleDim, st).Render(line)
}
