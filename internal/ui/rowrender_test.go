package ui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

func TestRelativeTime_tiersAndBoundaries(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	cases := []struct {
		name string
		ago  time.Duration
		want string
	}{
		{"seconds ago", 30 * time.Second, "now"},
		{"minutes", 10 * time.Minute, "10m"},
		{"just under an hour", 59 * time.Minute, "59m"},
		{"hours", 2 * time.Hour, "2h"},
		{"just under a day", 23 * time.Hour, "23h"},
		{"days", 12 * 24 * time.Hour, "12d"},
		{"exactly 30 days stays in days", 30 * 24 * time.Hour, "30d"},
		{"31 days switches to months", 31 * 24 * time.Hour, "1mo"},
		{"a few months", 100 * 24 * time.Hour, "3mo"},
		{"just under six months", 179 * 24 * time.Hour, "5mo"},
		{"six months caps", 180 * 24 * time.Hour, "6mo"},
		{"far beyond the cap", 700 * 24 * time.Hour, "6mo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := RelativeTime(now.Add(-tc.ago), now); got != tc.want {
				t.Errorf("RelativeTime(-%v) = %q, want %q", tc.ago, got, tc.want)
			}
		})
	}
}

func TestRelativeTime_zeroTimeRendersPlaceholder(t *testing.T) {
	t.Parallel()
	if got := RelativeTime(time.Time{}, time.Unix(2_000_000_000, 0)); got != "-" {
		t.Errorf("RelativeTime(zero) = %q, want -", got)
	}
}

func TestRenderLeafLine_longBranchTruncatesAndKeepsRightEdge(t *testing.T) {
	t.Parallel()
	w := domain.Worktree{
		Branch:         strings.Repeat("very-long-segment-", 10),
		Dirty:          true,
		LastCommitTime: time.Unix(1_999_000_000, 0),
	}
	const width = 60
	line := renderLeafLine("", w, strings.Repeat("very-long-segment-", 10), width, time.Unix(2_000_000_000, 0), rowState{}, iconsUnicode)

	if got := lipgloss.Width(line); got != width {
		t.Errorf("line width = %d, want %d", got, width)
	}
	if !strings.Contains(line, "11d") {
		t.Errorf("time should survive truncation: %q", line)
	}
	if !strings.Contains(line, "…") {
		t.Errorf("branch name should show an ellipsis: %q", line)
	}
	if !strings.Contains(line, iconsUnicode.dirty) {
		t.Errorf("the tag area is reserved; the dirty glyph must survive: %q", line)
	}
}

func TestRenderLeafLine_tagsSurviveLongPrefixNameAndBranch(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	w := domain.Worktree{
		Branch: strings.Repeat("very-long/", 12) + "leaf",
		Dirty:  true, Merged: true, UnpushedCount: 12, HasUpstream: true,
		LastCommitTime: now.Add(-200 * 24 * time.Hour),
	}
	const width = 60
	line := renderLeafLine(strings.Repeat("deep/repo/path/", 4)+" › ", w, strings.Repeat("n", 50), width, now, rowState{}, iconsUnicode)
	if got := lipgloss.Width(line); got != width {
		t.Errorf("line width = %d, want %d", got, width)
	}
	for _, glyph := range []string{iconsUnicode.dirty, iconsUnicode.merged, iconsUnicode.unpushed + "12"} {
		if !strings.Contains(line, glyph) {
			t.Errorf("tag %q must survive a crowded left side: %q", glyph, line)
		}
	}
	if !strings.Contains(line, "6mo") {
		t.Errorf("time column must survive: %q", line)
	}
}

func TestRenderLeafLine_cappedAgeCarriesWarningColor(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	old := domain.Worktree{Branch: "x", LastCommitTime: now.Add(-200 * 24 * time.Hour)}
	fresh := domain.Worktree{Branch: "x", LastCommitTime: now.Add(-time.Hour)}
	oldLine := renderLeafLine("", old, "x", 60, now, rowState{}, iconsUnicode)
	freshLine := renderLeafLine("", fresh, "x", 60, now, rowState{}, iconsUnicode)
	if !strings.Contains(oldLine, "\x1b[33m") {
		t.Errorf("6mo+ should carry the warning color: %q", oldLine)
	}
	if strings.Contains(freshLine, "\x1b[33m") {
		t.Errorf("a fresh timestamp should stay dim: %q", freshLine)
	}
}

func TestRenderLeafLine_tagsDoNotBreakAlignment(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	plain := domain.Worktree{Branch: "x", LastCommitTime: now.Add(-time.Hour)}
	tagged := domain.Worktree{
		Branch: "x", Dirty: true, Locked: true, Merged: true,
		UnpushedCount: 2, HasUpstream: true,
		LastCommitTime: now.Add(-time.Hour),
	}
	const width = 60
	for _, ic := range []iconSet{iconsNerd, iconsUnicode} {
		a := renderLeafLine("", plain, "x", width, now, rowState{}, ic)
		b := renderLeafLine("", tagged, "x", width, now, rowState{}, ic)
		if lipgloss.Width(a) != width || lipgloss.Width(b) != width {
			t.Errorf("widths = %d and %d, want both %d", lipgloss.Width(a), lipgloss.Width(b), width)
		}
	}
}

func TestRenderLeafLine_focusedAndSelectedKeepWidth(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	w := domain.Worktree{Branch: "x", Merged: true, LastCommitTime: now.Add(-time.Hour)}
	const width = 60
	for _, st := range []rowState{
		{focused: true},
		{selected: true},
		{inRange: true},
		{focused: true, selected: true, inRange: true},
	} {
		line := renderLeafLine("", w, "x", width, now, st, iconsUnicode)
		if got := lipgloss.Width(line); got != width {
			t.Errorf("state %+v: width = %d, want %d", st, got, width)
		}
	}
}

func TestRenderLeafLine_selectedShowsCheckMark(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	w := domain.Worktree{Branch: "x", LastCommitTime: now.Add(-time.Hour)}
	line := renderLeafLine("", w, "x", 60, now, rowState{selected: true}, iconsUnicode)
	if !strings.Contains(line, "✓") {
		t.Errorf("selected leaf should carry ✓: %q", line)
	}
}

func TestRenderLeafLine_selectionShowsAsRowBackground(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	w := domain.Worktree{Branch: "x", LastCommitTime: now.Add(-time.Hour)}
	selected := renderLeafLine("", w, "x", 60, now, rowState{selected: true}, iconsUnicode)
	plain := renderLeafLine("", w, "x", 60, now, rowState{}, iconsUnicode)
	if selected == plain {
		t.Fatal("selected row should render differently from an unselected one")
	}
	if !strings.Contains(selected, "48;") {
		t.Errorf("selected row should carry a background color: %q", selected)
	}
	if strings.Contains(plain, "48;") {
		t.Errorf("unselected row should carry no background color: %q", plain)
	}
}

func TestRenderLeafLine_unselectedCarriesNoPlaceholderMark(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	for _, w := range []domain.Worktree{
		{Branch: "x", LastCommitTime: now.Add(-time.Hour)},
		{Branch: "main", IsMain: true, LastCommitTime: now.Add(-time.Hour)},
	} {
		line := renderLeafLine("", w, w.Branch, 60, now, rowState{}, iconsUnicode)
		plain := stripANSI(line)
		if strings.Contains(plain, "·") || strings.Contains(plain, "–") {
			t.Errorf("unselected leaf should carry no mark: %q", plain)
		}
	}
}

func TestRenderLeafLine_branchAnnotationFollowsName(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	w := domain.Worktree{Branch: "feature/x", Path: "/wt/x", LastCommitTime: now.Add(-time.Hour)}
	line := stripANSI(renderLeafLine("", w, "x", 60, now, rowState{}, iconsUnicode))
	if !strings.Contains(line, "x (feature/x)") {
		t.Errorf("leaf should carry the full branch in parens: %q", line)
	}
}

func TestRenderLeafLine_detachedAnnotationShowsSha(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	w := domain.Worktree{
		HeadSHA:        "abcdef0123456789abcdef0123456789abcdef01",
		Path:           "/wt/x",
		LastCommitTime: now.Add(-time.Hour),
	}
	line := stripANSI(renderLeafLine("", w, "x", 60, now, rowState{}, iconsUnicode))
	if !strings.Contains(line, "(detached: abcdef0)") {
		t.Errorf("detached leaf should carry the short SHA: %q", line)
	}
}

func TestRenderPathLine_prunableShowsMissingAnnotation(t *testing.T) {
	t.Parallel()
	w := domain.Worktree{Branch: "x", Path: "/gone/x", Prunable: true}
	line := renderPathLine("", w, 60, rowState{})
	if !strings.Contains(line, "(missing)") {
		t.Errorf("prunable path line should carry (missing): %q", line)
	}
	if !strings.Contains(line, "/gone/x") {
		t.Errorf("path line should carry the path: %q", line)
	}
}

func TestTagParts_protectedAndStatusGlyphs(t *testing.T) {
	t.Parallel()
	w := domain.Worktree{
		IsMain: true, Dirty: true, Locked: true, Merged: true,
		UnpushedCount: 3, HasUpstream: true, Prunable: true,
	}
	for _, ic := range []iconSet{iconsNerd, iconsUnicode} {
		var joined strings.Builder
		for _, p := range tagParts(w, ic, false) {
			joined.WriteString(p.text + " ")
		}
		tags := joined.String()
		for _, glyph := range []string{ic.main, ic.dirty, ic.locked, ic.merged, ic.prunable} {
			if !strings.Contains(tags, glyph) {
				t.Errorf("tags %q missing glyph %q", tags, glyph)
			}
		}
		if !strings.Contains(tags, ic.unpushed+"3") {
			t.Errorf("tags %q should carry the unpushed count %q", tags, ic.unpushed+"3")
		}
	}
	if len(tagParts(domain.Worktree{}, iconsUnicode, false)) != 0 {
		t.Error("clean worktree should carry no tags")
	}
}

func TestTagParts_glyphsCarryNoTextLabels(t *testing.T) {
	t.Parallel()
	w := domain.Worktree{Dirty: true, Locked: true, Merged: true, Prunable: true}
	for _, p := range tagParts(w, iconsUnicode, false) {
		for _, label := range []string{"dirty", "locked", "merged", "gone"} {
			if strings.Contains(p.text, label) {
				t.Errorf("tag %q should be glyph-only (legend lives in help)", p.text)
			}
		}
	}
}

func TestTagParts_mutedShowsOnlyHomeMarker(t *testing.T) {
	t.Parallel()
	w := domain.Worktree{
		IsMain: true, IsCurrent: true, Dirty: true, Merged: true,
		UnpushedCount: 2, HasUpstream: true,
	}
	tags := tagParts(w, iconsUnicode, true)
	if len(tags) != 1 || tags[0].text != iconsUnicode.main {
		t.Errorf("muted main worktree should carry only the home marker, got %+v", tags)
	}
	if got := tagParts(domain.Worktree{Dirty: true}, iconsUnicode, true); len(got) != 0 {
		t.Errorf("muted non-main worktree should carry no tags, got %+v", got)
	}
}

func TestRenderLeafLine_mutedGraysNameAndAnnotation(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	w := domain.Worktree{Branch: "main", Path: "/wt/main", LastCommitTime: now.Add(-time.Hour)}
	muted := renderLeafLine("", w, "main", 60, now, rowState{muted: true}, iconsUnicode)
	normal := renderLeafLine("", w, "main", 60, now, rowState{}, iconsUnicode)
	if muted == normal {
		t.Error("the default branch should render dimmer than other worktrees")
	}
	if !strings.Contains(muted, "\x1b[2m") {
		t.Errorf("muted entry should carry the faint attribute: %q", muted)
	}
}

func TestIconSetByName_validatesNames(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"", "nerd", "unicode"} {
		if _, ok := iconSetByName(name); !ok {
			t.Errorf("iconSetByName(%q) should be valid", name)
		}
	}
	if _, ok := iconSetByName("emoji"); ok {
		t.Error("iconSetByName(emoji) should be rejected")
	}
}

func TestPathConnectorPrefix_continuesGuidesThroughPathLines(t *testing.T) {
	t.Parallel()
	cases := []struct {
		isLast []bool
		want   string
	}{
		// A leaf with later siblings keeps the bar running through its
		// path line; the last child leaves a gap (ADR 0013).
		{[]bool{false}, "│   "},
		{[]bool{true}, "    "},
		{[]bool{false, false}, "│   │   "},
		{[]bool{false, true}, "│       "},
		{[]bool{true, false}, "    │   "},
	}
	for _, tc := range cases {
		if got := pathConnectorPrefix(tc.isLast); got != tc.want {
			t.Errorf("pathConnectorPrefix(%v) = %q, want %q", tc.isLast, got, tc.want)
		}
	}
}

func TestConnectorPrefix_deriveFromIsLast(t *testing.T) {
	t.Parallel()
	cases := []struct {
		isLast []bool
		want   string
	}{
		{[]bool{false}, "├── "},
		{[]bool{true}, "└── "},
		{[]bool{false, true}, "│   └── "},
		{[]bool{true, false}, "    ├── "},
	}
	for _, tc := range cases {
		if got := connectorPrefix(tc.isLast); got != tc.want {
			t.Errorf("connectorPrefix(%v) = %q, want %q", tc.isLast, got, tc.want)
		}
	}
}
