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

// viewHelp renders the keybinding cheat sheet.
func (m Model) viewHelp() string {
	var b strings.Builder
	b.WriteString("Keybindings\n\n")
	rows := []key.Binding{
		keys.Up, keys.Down, keys.Collapse, keys.Expand,
		keys.Select, keys.Visual, keys.Delete, keys.Escape,
		keys.Filter, keys.Merged, keys.Sort, keys.Help, keys.Quit,
	}
	for _, binding := range rows {
		h := binding.Help()
		fmt.Fprintf(&b, "  %-10s %s\n", h.Key, h.Desc)
	}
	b.WriteString("\n" + styleDim.Render("press any key to close"))
	return b.String()
}
