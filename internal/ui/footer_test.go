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
	if f := m.footer(); !strings.Contains(f, "1 selected") || !strings.Contains(f, "delete") {
		t.Errorf("selection footer should steer toward delete: %q", f)
	}

	m = apply(t, m, press('v'))
	if f := m.footer(); !strings.Contains(f, "VISUAL") || !strings.Contains(f, "extend") {
		t.Errorf("visual footer should show the badge and extend hint: %q", f)
	}
}

func TestStatusLine_showsVisualBadgeAndSelection(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space())
	if s := m.statusLine(); !strings.Contains(s, "1 selected") {
		t.Errorf("status line should count the selection: %q", s)
	}
	m = apply(t, m, press('v'))
	if s := m.statusLine(); !strings.Contains(s, "VISUAL") {
		t.Errorf("status line should show the visual badge: %q", s)
	}
}
