package ui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/action"
)

// DeleteResultsMsg carries the batch outcome back into the UI.
type DeleteResultsMsg struct {
	Results []action.Result
}

// buildPlan turns the current selection into ordered plan items with branch
// deletion on by default.
func (m *Model) buildPlan() []action.PlanItem {
	var items []action.PlanItem
	for i := range m.repos {
		repo := &m.repos[i]
		for _, w := range repo.Worktrees {
			if m.selection[w.Path] {
				items = append(items, action.PlanItem{
					RepoPath:     repo.Path,
					Worktree:     w,
					DeleteBranch: !w.Detached(),
				})
			}
		}
	}
	return items
}

// updateConfirmKey handles input on the confirmation screen.
func (m Model) updateConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.confirmCursor < len(m.confirmItems)-1 {
			m.confirmCursor++
		}
	case "k", "up":
		if m.confirmCursor > 0 {
			m.confirmCursor--
		}
	case "space":
		it := &m.confirmItems[m.confirmCursor]
		if !it.Worktree.Detached() {
			it.DeleteBranch = !it.DeleteBranch
		}
	case "y":
		items := m.confirmItems
		m.phase = phaseExecuting
		return m, func() tea.Msg {
			return DeleteResultsMsg{Results: action.Execute(context.Background(), items)}
		}
	case "n", "q", "esc":
		m.phase = phaseBrowsing
		m.confirmItems = nil
	}
	return m, nil
}

// viewConfirm renders the batch confirmation screen.
func (m Model) viewConfirm() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Delete %d worktree(s)?\n", len(m.confirmItems))
	b.WriteString(styleDim.Render("space: toggle branch deletion for the highlighted item") + "\n\n")
	for i, it := range m.confirmItems {
		cursor := "  "
		if i == m.confirmCursor {
			cursor = "> "
		}
		branchMark := "[x] delete branch"
		if it.Worktree.Detached() {
			branchMark = "    (detached: no branch)"
		} else if !it.DeleteBranch {
			branchMark = "[ ] keep branch"
		}
		label := it.Worktree.Branch
		if it.Worktree.Detached() {
			label = leafLabel(it)
		}
		fmt.Fprintf(&b, "%s%s  %s\n", cursor, label, branchMark)
		fmt.Fprintf(&b, "    %s\n", styleDim.Render(it.Worktree.Path))
		for _, warn := range warningsFor(it) {
			fmt.Fprintf(&b, "    %s\n", warn)
		}
	}
	b.WriteString("\n" + styleDim.Render("y confirm  ·  n/esc cancel"))
	return b.String()
}

// leafLabel names a detached-HEAD item in the confirmation list.
func leafLabel(it action.PlanItem) string {
	sha := it.Worktree.HeadSHA
	if len(sha) > 7 {
		sha = sha[:7]
	}
	return fmt.Sprintf("(detached: %s)", sha)
}

// warningsFor lists the force-related warnings of one item (ADR 0005: warn,
// never block).
func warningsFor(it action.PlanItem) []string {
	var warns []string
	if it.Worktree.Dirty {
		warns = append(warns, "⚠ has uncommitted changes")
	}
	if it.Worktree.Locked {
		warns = append(warns, "⚠ locked (will force-unlock)")
	}
	if it.Worktree.UnpushedCount > 0 {
		warns = append(warns, fmt.Sprintf("⚠ %d commit(s) not pushed to the remote", it.Worktree.UnpushedCount))
	}
	return warns
}
