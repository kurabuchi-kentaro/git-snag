package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/version"
)

// WithAnimation returns a copy of the model with the deletion animation
// enabled or disabled (wired from config / --no-animation).
func (m Model) WithAnimation(enabled bool) Model {
	m.animationEnabled = enabled
	return m
}

// ValidIconSet reports whether name selects a known icon set, so the CLI
// can reject bad config/flag values before the TUI starts (REQ-A6).
func ValidIconSet(name string) bool {
	_, ok := iconSetByName(name)
	return ok
}

// WithIcons returns a copy of the model using the named icon set (wired
// from config / --icons). Unknown names keep the default; validate with
// ValidIconSet first.
func (m Model) WithIcons(name string) Model {
	if set, ok := iconSetByName(name); ok {
		m.icons = set
	}
	return m
}

// viewHelp renders the keybinding cheat sheet and the tag legend (rows show
// bare glyphs; their meaning lives here) side by side, so the modal stays
// within a 24-line terminal.
func (m Model) viewHelp() string {
	var kb strings.Builder
	kb.WriteString(styleKey.Render("Keybindings") + "\n\n")
	rows := []key.Binding{
		keys.Up, keys.Down, keys.Collapse, keys.Expand,
		keys.Select, keys.Visual, keys.Delete, keys.Escape,
		keys.Filter, keys.Merged, keys.Sort, keys.Info, keys.Branches,
		keys.Help, keys.Quit,
	}
	for _, binding := range rows {
		h := binding.Help()
		fmt.Fprintf(&kb, "%s %s\n", styleKey.Render(fmt.Sprintf("%-10s", h.Key)), styleDim.Render(h.Desc))
	}

	var tg strings.Builder
	tg.WriteString(styleKey.Render("Tags") + "\n\n")
	legend := []struct {
		glyph string
		style lipgloss.Style
		desc  string
	}{
		{m.icons.current, styleDim, "current worktree"},
		{m.icons.main, styleDim, "main worktree"},
		{m.icons.dirty, styleWarn, "dirty (uncommitted changes)"},
		{m.icons.locked, styleInfo, "locked"},
		{m.icons.merged, styleGood, "merged into the default branch"},
		{m.icons.unpushed + "n", styleSync, "n commits not on the remote"},
		{m.icons.prunable, styleBad, "gone (directory missing)"},
		{m.icons.gone, styleGood, "upstream deleted on the remote"},
	}
	for _, e := range legend {
		pad := strings.Repeat(" ", max(4-lipgloss.Width(e.glyph), 1))
		fmt.Fprintf(&tg, "%s%s%s\n", e.style.Render(e.glyph), pad, styleDim.Render(e.desc))
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, kb.String(), "    ", tg.String())
	return body + "\n" + styleDim.Render("git-snag "+version.Version+" · press any key to close")
}
