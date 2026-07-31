package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestOverlayModal_centersContentOverBase(t *testing.T) {
	t.Parallel()
	base := strings.TrimSuffix(strings.Repeat("bbbbbbbbbbbbbbbbbbbb\n", 10), "\n")
	out := overlayModal(base, "hello", 20, 10, 0, 0)
	if !strings.Contains(out, "hello") {
		t.Errorf("overlay should contain the modal content:\n%s", out)
	}
	if got := len(strings.Split(out, "\n")); got != 10 {
		t.Errorf("overlay height = %d lines, want 10", got)
	}
}

func TestOverlayModal_degradesGracefullyOnZeroSize(t *testing.T) {
	t.Parallel()
	if got := overlayModal("base", "modal", 0, 0, 0, 0); got != "base" {
		t.Errorf("zero-size overlay should return the base view, got %q", got)
	}
}

func TestModalInteriorWidth_twoThirdsCappedAt100(t *testing.T) {
	t.Parallel()
	if got := modalInteriorWidth(200); got != 100 {
		t.Errorf("width for a wide terminal = %d, want 100", got)
	}
	if got := modalInteriorWidth(120); got != 80 {
		t.Errorf("width for a 120-col terminal = %d, want 80", got)
	}
	if got := modalInteriorWidth(60); got != 40 {
		t.Errorf("width for a 60-col terminal = %d, want 40", got)
	}
	if got := modalInteriorWidth(10); got != 20 {
		t.Errorf("width floor = %d, want 20", got)
	}
}

// stripANSI removes escape sequences so position assertions see the plain
// cell grid.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case inEsc:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
		case r == '\x1b':
			inEsc = true
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestOverlayModal_keepsBackdropVisibleAndCentersFrame(t *testing.T) {
	t.Parallel()
	const width, height = 100, 30
	baseLine := strings.Repeat("B", width)
	base := strings.TrimSuffix(strings.Repeat(baseLine+"\n", height), "\n")
	out := stripANSI(overlayModal(base, "MODAL-CONTENT", width, height, 0, 0))
	lines := strings.Split(out, "\n")

	if !strings.Contains(out, "BBBB") {
		t.Fatalf("dimmed backdrop should stay visible around the modal:\n%s", out)
	}
	found := -1
	for i, line := range lines {
		if idx := strings.Index(line, "MODAL-CONTENT"); idx >= 0 {
			found = i
			if idx < width/4 {
				t.Errorf("modal content starts at column %d; expected roughly centered", idx)
			}
		}
	}
	if found < 0 {
		t.Fatalf("modal content missing:\n%s", out)
	}
	if found < height/4 || found > height*3/4 {
		t.Errorf("modal content on line %d of %d; expected roughly vertically centered", found, height)
	}
}

func TestWindowLines_padsShortContentToFixedHeight(t *testing.T) {
	t.Parallel()
	got := windowLines([]string{"a", "b"}, 0, 4)
	if len(got) != 4 {
		t.Fatalf("window = %d lines, want 4", len(got))
	}
	if got[0] != "a" || got[3] != "" {
		t.Errorf("window = %v", got)
	}
}

func TestWindowLines_clampsOffset(t *testing.T) {
	t.Parallel()
	lines := []string{"a", "b", "c", "d", "e"}
	got := windowLines(lines, 99, 2)
	if got[0] != "d" || got[1] != "e" {
		t.Errorf("over-scrolled window = %v, want the last two lines", got)
	}
}

func TestView_confirmModalRendersOverTree(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space(), press('d'))
	view := m.View().Content
	if !strings.Contains(view, "Delete 1 worktree(s)?") {
		t.Errorf("confirm modal missing from view:\n%s", view)
	}
	if got := len(strings.Split(view, "\n")); got > 24 {
		t.Errorf("overlaid view = %d lines, want at most the 24-line terminal", got)
	}
}

func TestSizeModal_fixesActFrameAcrossPhases(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space(), press('d'))
	if m.modalW <= 0 || m.modalH <= 0 {
		t.Fatalf("modal dims = %dx%d, want positive after opening confirm", m.modalW, m.modalH)
	}
	if lipgloss.Height(m.viewConfirm()) != m.modalH {
		t.Errorf("confirm act height = %d, want %d", lipgloss.Height(m.viewConfirm()), m.modalH)
	}
	if lipgloss.Height(m.viewWaiting()) != m.modalH {
		t.Errorf("waiting act height = %d, want %d", lipgloss.Height(m.viewWaiting()), m.modalH)
	}
}
