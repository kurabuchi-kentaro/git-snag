package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
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

// viewHelp renders the keybinding cheat sheet as modal content.
func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString(styleKey.Render("Keybindings") + "\n\n")
	rows := []key.Binding{
		keys.Up, keys.Down, keys.Collapse, keys.Expand,
		keys.Select, keys.Visual, keys.Delete, keys.Escape,
		keys.Filter, keys.Merged, keys.Sort, keys.Help, keys.Quit,
	}
	for _, binding := range rows {
		h := binding.Help()
		fmt.Fprintf(&b, "%s %s\n", styleKey.Render(fmt.Sprintf("%-10s", h.Key)), styleDim.Render(h.Desc))
	}
	b.WriteString("\n" + styleDim.Render("press any key to close"))
	return b.String()
}
