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
		{"minutes", 10 * time.Minute, "10m ago"},
		{"just under an hour", 59 * time.Minute, "59m ago"},
		{"hours", 2 * time.Hour, "2h ago"},
		{"just under a day", 23 * time.Hour, "23h ago"},
		{"days", 12 * 24 * time.Hour, "12d ago"},
		{"exactly 30 days stays in days", 30 * 24 * time.Hour, "30d ago"},
		{"31 days switches to months", 31 * 24 * time.Hour, "1mo ago"},
		{"a few months", 100 * 24 * time.Hour, "3mo ago"},
		{"just under six months", 179 * 24 * time.Hour, "5mo ago"},
		{"six months caps", 180 * 24 * time.Hour, "6mo+ ago"},
		{"far beyond the cap", 700 * 24 * time.Hour, "6mo+ ago"},
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
	if !strings.Contains(line, "ago") {
		t.Errorf("time should survive truncation: %q", line)
	}
	if !strings.Contains(line, "…") {
		t.Errorf("branch name should show an ellipsis: %q", line)
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

func TestRenderLeafLine_mutedNameRendersFaint(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	w := domain.Worktree{Branch: "dev", LastCommitTime: now.Add(-time.Hour)}
	muted := renderLeafLine("", w, "dev", 60, now, rowState{muted: true}, iconsUnicode)
	normal := renderLeafLine("", w, "dev", 60, now, rowState{}, iconsUnicode)
	if muted == normal {
		t.Error("the default branch should render dimmer than other branches")
	}
	if !strings.Contains(muted, "\x1b[2m") {
		t.Errorf("muted name should carry the faint attribute: %q", muted)
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
		for _, p := range tagParts(w, ic) {
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
	if len(tagParts(domain.Worktree{}, iconsUnicode)) != 0 {
		t.Error("clean worktree should carry no tags")
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
