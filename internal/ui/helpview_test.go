package ui

import (
	"strings"
	"testing"
)

func TestHelp_opensAndClosesWithAnyKey(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	m = apply(t, m, press('?'))
	if !m.showHelp {
		t.Fatal("? should open help")
	}
	view := m.View().Content
	for _, want := range []string{"Keybindings", "visual select", "delete selected", "merged only"} {
		if !strings.Contains(view, want) {
			t.Errorf("help view missing %q:\n%s", want, view)
		}
	}
	m = apply(t, m, press('x'))
	if m.showHelp {
		t.Error("any key should close help")
	}
	if m.phase != phaseBrowsing {
		t.Errorf("closing help should not change phase: %v", m.phase)
	}
}

func TestWithAnimation_togglesTheFlag(t *testing.T) {
	t.Parallel()
	if NewModel().WithAnimation(false).animationEnabled {
		t.Error("WithAnimation(false) should disable the animation")
	}
	if !NewModel().WithAnimation(true).animationEnabled {
		t.Error("WithAnimation(true) should enable the animation")
	}
}
