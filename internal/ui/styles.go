// Package ui implements the Bubble Tea terminal interface. It never imports
// internal/gitcli or os/exec: all data arrives as tea.Msg values and all
// mutation happens through internal/action commands.
package ui

import "charm.land/lipgloss/v2"

// Status icons shown on worktree rows.
const (
	iconDirty    = "📝"
	iconLocked   = "🔒"
	iconMerged   = "✅"
	iconUnpushed = "⬆️"
	iconPrunable = "👻"
	iconCurrent  = "📍"
	iconMain     = "🏠"
)

var (
	styleRepoName  = lipgloss.NewStyle().Bold(true)
	styleDim       = lipgloss.NewStyle().Faint(true)
	styleFocused   = lipgloss.NewStyle().Reverse(true)
	styleProtected = lipgloss.NewStyle().Faint(true)
)
