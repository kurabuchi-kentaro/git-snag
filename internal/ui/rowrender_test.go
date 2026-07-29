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
	line := renderLeafLine("", w, strings.Repeat("very-long-segment-", 10), width, time.Unix(2_000_000_000, 0), false, false)

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

func TestRenderLeafLine_emojiTagsDoNotBreakAlignment(t *testing.T) {
	t.Parallel()
	now := time.Unix(2_000_000_000, 0)
	plain := domain.Worktree{Branch: "x", LastCommitTime: now.Add(-time.Hour)}
	tagged := domain.Worktree{
		Branch: "x", Dirty: true, Locked: true, Merged: true,
		UnpushedCount: 2, HasUpstream: true,
		LastCommitTime: now.Add(-time.Hour),
	}
	const width = 60
	a := renderLeafLine("", plain, "x", width, now, false, false)
	b := renderLeafLine("", tagged, "x", width, now, false, false)
	if lipgloss.Width(a) != width || lipgloss.Width(b) != width {
		t.Errorf("widths = %d and %d, want both %d", lipgloss.Width(a), lipgloss.Width(b), width)
	}
}

func TestRenderPathLine_prunableShowsMissingAnnotation(t *testing.T) {
	t.Parallel()
	w := domain.Worktree{Branch: "x", Path: "/gone/x", Prunable: true}
	line := renderPathLine("", w, 60)
	if !strings.Contains(line, "(missing)") {
		t.Errorf("prunable path line should carry (missing): %q", line)
	}
	if !strings.Contains(line, "/gone/x") {
		t.Errorf("path line should carry the path: %q", line)
	}
}

func TestTagsFor_protectedAndStatusIcons(t *testing.T) {
	t.Parallel()
	w := domain.Worktree{
		IsMain: true, Dirty: true, Locked: true, Merged: true,
		UnpushedCount: 1, HasUpstream: true, Prunable: true,
	}
	tags := tagsFor(w)
	for _, icon := range []string{iconMain, iconDirty, iconLocked, iconMerged, iconUnpushed, iconPrunable} {
		if !strings.Contains(tags, icon) {
			t.Errorf("tags %q missing icon %q", tags, icon)
		}
	}
	if strings.Contains(tagsFor(domain.Worktree{}), iconDirty) {
		t.Error("clean worktree should carry no dirty icon")
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
