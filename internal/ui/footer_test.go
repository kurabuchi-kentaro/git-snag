package ui

import (
	"strings"
	"testing"
)

func TestFooter_swapsHintsWithContext(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"))

	if f := m.footer(); !strings.Contains(f, "select") || strings.Contains(f, "delete") {
		t.Errorf("browsing footer should offer select, not delete: %q", f)
	}

	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space())
	f := m.footer()
	if !strings.Contains(f, "delete") {
		t.Errorf("selection footer should steer toward delete: %q", f)
	}
	// The count lives in the status line, not the footer.
	if strings.Contains(f, "selected") {
		t.Errorf("footer should not repeat the selection count: %q", f)
	}

	m = apply(t, m, press('v'))
	if f := m.footer(); !strings.Contains(f, "VISUAL") || !strings.Contains(f, "extend") {
		t.Errorf("visual footer should show the badge and extend hint: %q", f)
	}
}

func TestStatusLine_countsMatchTheView(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("solo"), testRepo("multi", "feature/x"))
	if s := m.statusLine(); !strings.Contains(s, "1 repos") || !strings.Contains(s, "2 worktrees") {
		t.Errorf("counts should exclude the hidden single-worktree repo: %q", s)
	}
	m = apply(t, m, press('i')) // reveal solo
	if s := m.statusLine(); !strings.Contains(s, "2 repos") || !strings.Contains(s, "3 worktrees") {
		t.Errorf("counts should include revealed repos: %q", s)
	}
	m = apply(t, m, press('s')) // flat: candidates only
	if s := m.statusLine(); !strings.Contains(s, "1 repos") || !strings.Contains(s, "1 worktrees") {
		t.Errorf("flat-mode counts should match the candidate list: %q", s)
	}
}

func TestStatusLine_showsSelectionButNoVisualBadge(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space())
	if s := m.statusLine(); !strings.Contains(s, "selected") {
		t.Errorf("status line should reflect the selection: %q", s)
	}
	// The footer already carries the VISUAL badge; the status line must not
	// repeat it.
	m = apply(t, m, press('v'))
	if s := m.statusLine(); strings.Contains(s, "VISUAL") {
		t.Errorf("status line should not duplicate the visual badge: %q", s)
	}
}

func TestStatusLine_selectionRendersAsFractions(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a", "feature/b"), testRepo("beta", "feature/c"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space())
	s := stripANSI(m.statusLine())
	// One of two repos touched, one of five visible worktrees selected.
	if !strings.Contains(s, "1/2 repos") || !strings.Contains(s, "1/5 worktrees selected") {
		t.Errorf("status line should show selection fractions: %q", s)
	}
	m = apply(t, m, esc())
	if s := stripANSI(m.statusLine()); strings.Contains(s, "/") || strings.Contains(s, "selected") {
		t.Errorf("cleared selection should restore plain counts: %q", s)
	}
}
