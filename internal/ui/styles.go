// Package ui implements the Bubble Tea terminal interface. It never imports
// internal/gitcli or os/exec: all data arrives as tea.Msg values and all
// mutation happens through internal/action commands.
package ui

import "charm.land/lipgloss/v2"

// The palette follows ADR 0013: state colors come from the terminal's ANSI
// palette so they track the user's theme; the one fixed color is the purple
// accent, which marks operations (selection, visual range, filter, modal
// frames) and degrades to ANSI magenta on non-truecolor terminals.
var (
	colorAccent = lipgloss.Color("#9d7cd8") // operations: selection, visual, modals
	colorGood   = lipgloss.Color("2")       // merged, success
	colorWarn   = lipgloss.Color("3")       // dirty, warnings
	colorInfo   = lipgloss.Color("4")       // locked; directory groups
	colorBad    = lipgloss.Color("1")       // prunable, failures, danger titles
	colorSync   = lipgloss.Color("6")       // unpushed (remote out of sync)
	colorFocus  = lipgloss.Color("8")       // focused-row background (theme gray)
	colorSelBg  = lipgloss.Color("#22402c") // selected-row background (dark green: picked, check-marked)
	colorRule   = lipgloss.Color("#2a2440") // header/footer outline (subdued accent-family dark)
)

var (
	// Repo headers stay colorless: purple now means branches and blue means
	// directories/worktrees, so the header reads as a bold section title.
	styleRepoName   = lipgloss.NewStyle().Bold(true)
	styleWorktree   = lipgloss.NewStyle().Foreground(colorInfo)
	styleGroup      = lipgloss.NewStyle().Foreground(colorInfo).Bold(true)
	styleDim        = lipgloss.NewStyle().Faint(true)
	styleRule       = lipgloss.NewStyle().Foreground(colorRule)
	styleAccent     = lipgloss.NewStyle().Foreground(colorAccent)
	styleAccentBold = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	styleKey        = lipgloss.NewStyle().Bold(true)
	styleBadge      = lipgloss.NewStyle().Foreground(lipgloss.Color("#f5f2fa")).Background(colorAccent).Bold(true)
	// styleBadgeInfo marks persistent view modes (blue), as opposed to the
	// purple styleBadge for transient gestures like visual mode (ADR 0014).
	styleBadgeInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("#f5f2fa")).Background(colorInfo).Bold(true)
	styleGood      = lipgloss.NewStyle().Foreground(colorGood)
	styleGoodBold  = lipgloss.NewStyle().Foreground(colorGood).Bold(true)
	styleWarn      = lipgloss.NewStyle().Foreground(colorWarn)
	styleInfo      = lipgloss.NewStyle().Foreground(colorInfo)
	styleBad       = lipgloss.NewStyle().Foreground(colorBad)
	styleSync      = lipgloss.NewStyle().Foreground(colorSync)
	styleDanger    = lipgloss.NewStyle().Foreground(colorBad).Bold(true)
	styleModal     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).Padding(0, 1)
)

// gutterBar marks rows inside the visual-mode range in the one-cell gutter
// every row reserves at the left edge.
const gutterBar = "▌"

// iconSet is one table of status glyphs. Both sets are single-width,
// colorable glyphs (ADR 0013); emoji are gone.
type iconSet struct {
	dirty    string
	locked   string
	merged   string
	unpushed string // prefix of the count: "↑3"
	prunable string
	current  string
	main     string
	gone     string // upstream deleted on the remote (branch mode)
}

// iconsNerd is the default set and assumes a Nerd Font patched terminal
// font. iconsUnicode is the fallback selected via config or --icons.
var (
	iconsNerd = iconSet{
		dirty:    "", // nf-fa-pencil
		locked:   "", // nf-fa-lock
		merged:   "", // nf-oct-git_merge
		unpushed: "↑",
		prunable: "\U000f02a0", // nf-md-ghost
		current:  "",          // nf-fa-map_marker
		main:     "",          // nf-fa-home
		gone:     "",          // nf-fa-chain_broken
	}
	iconsUnicode = iconSet{
		dirty:    "±",
		locked:   "⊘",
		merged:   "✓",
		unpushed: "↑",
		prunable: "†",
		current:  "●",
		main:     "⌂",
		gone:     "↯",
	}
)

// iconSetByName maps the config/flag value to a glyph table; unknown names
// report ok=false so the CLI can fail loudly (REQ-A6).
func iconSetByName(name string) (iconSet, bool) {
	switch name {
	case "", "nerd":
		return iconsNerd, true
	case "unicode":
		return iconsUnicode, true
	default:
		return iconSet{}, false
	}
}
