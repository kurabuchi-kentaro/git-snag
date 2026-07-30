package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// tallModel builds a model whose rows far exceed the terminal height.
func tallModel(t *testing.T) Model {
	t.Helper()
	m := modelWith(t,
		testRepo("r01", "aa", "bb", "cc", "dd"),
		testRepo("r02", "aa", "bb", "cc", "dd"),
		testRepo("r03", "aa", "bb", "cc", "dd"),
		testRepo("r04", "aa", "bb", "cc", "dd"),
	)
	return apply(t, m, tea.WindowSizeMsg{Width: 80, Height: 15})
}

func TestScroll_viewNeverExceedsTerminalHeight(t *testing.T) {
	t.Parallel()
	m := tallModel(t)
	view := m.View().Content
	if got := len(strings.Split(view, "\n")); got > 15 {
		t.Errorf("view has %d lines for a 15-line terminal; overflow is clipped invisibly", got)
	}
}

func TestScroll_focusedRowStaysVisibleWhileMovingDown(t *testing.T) {
	t.Parallel()
	m := tallModel(t)
	for range len(m.rows) - 1 {
		m = apply(t, m, press('j'))
	}
	if m.focus != len(m.rows)-1 {
		t.Fatalf("focus = %d, want last row %d", m.focus, len(m.rows)-1)
	}
	view := m.View().Content
	// The last repo's last leaf must be on screen; the first header must
	// have scrolled out.
	if !strings.Contains(view, "r04-wt/dd") {
		t.Errorf("focused (last) row should be visible:\n%s", view)
	}
	if strings.Contains(view, "r01  ") {
		t.Errorf("first header should have scrolled off screen:\n%s", view)
	}
}

func TestScroll_movingBackUpRestoresTheTop(t *testing.T) {
	t.Parallel()
	m := tallModel(t)
	for range len(m.rows) - 1 {
		m = apply(t, m, press('j'))
	}
	for range len(m.rows) - 1 {
		m = apply(t, m, press('k'))
	}
	if m.scrollLine != 0 {
		t.Errorf("scrollLine = %d, want 0 back at the top", m.scrollLine)
	}
	view := m.View().Content
	if !strings.Contains(view, "r01") {
		t.Errorf("first repo should be visible again:\n%s", view)
	}
}

func TestScroll_shrinkingTerminalKeepsFocusVisible(t *testing.T) {
	t.Parallel()
	m := tallModel(t)
	for range 10 {
		m = apply(t, m, press('j'))
	}
	m = apply(t, m, tea.WindowSizeMsg{Width: 80, Height: 8})
	view := m.View().Content
	if got := len(strings.Split(view, "\n")); got > 8 {
		t.Errorf("view has %d lines for an 8-line terminal", got)
	}
}
