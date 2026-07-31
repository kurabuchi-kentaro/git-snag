package ui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/action"
)

// DeleteResultsMsg carries the batch outcome back into the UI.
type DeleteResultsMsg struct {
	Results []action.Result
}

// buildPlan turns the current selection into ordered plan items with branch
// deletion on by default. In branch mode a selected branch with a worktree
// becomes the normal worktree-plus-branch item; a bare branch becomes a
// branch-only item (ADR 0014).
func (m *Model) buildPlan() []action.PlanItem {
	var items []action.PlanItem
	for i := range m.repos {
		repo := &m.repos[i]
		if m.branchMode {
			for _, b := range repo.Branches {
				if !m.selection[branchKey(repo.Path, b.Name)] {
					continue
				}
				if w := worktreeByPath(repo, b.WorktreePath); w != nil {
					items = append(items, action.PlanItem{
						RepoPath:     repo.Path,
						Worktree:     *w,
						DeleteBranch: true,
					})
				} else {
					items = append(items, action.PlanItem{RepoPath: repo.Path, Branch: b.Name})
				}
			}
			continue
		}
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
	case "j", keyDown:
		if m.confirmCursor < len(m.confirmItems)-1 {
			m.confirmCursor++
		}
	case "k", keyUp:
		if m.confirmCursor > 0 {
			m.confirmCursor--
		}
	case "space":
		it := &m.confirmItems[m.confirmCursor]
		if !it.BranchOnly() && !it.Worktree.Detached() {
			it.DeleteBranch = !it.DeleteBranch
		}
	case "y":
		items := m.confirmItems
		execute := func() tea.Msg {
			return DeleteResultsMsg{Results: action.Execute(context.Background(), items)}
		}
		if !m.animationEnabled {
			m.phase = phaseExecuting
			return m, execute
		}
		// Animation and deletion start together (REQ-P5); the summary waits
		// for whichever finishes last.
		m.phase = phaseExploding
		m.explosionFrame = 0
		m.explosionDone = false
		m.resultsArrived = false
		m.explosionTier = tierFor(len(items))
		m.explosionSeed = m.now().UnixNano()
		return m, tea.Batch(execute, explosionTick())
	case "n", "q", "esc":
		m.phase = phaseBrowsing
		m.confirmItems = nil
	}
	return m, nil
}

// sizeModal fixes the modal interior for the delete flow's three acts
// (confirm → explosion → summary) so the frame never jumps between them.
func (m *Model) sizeModal() {
	m.modalW = modalInteriorWidth(m.width)
	lines, _ := m.confirmLines(m.modalW)
	m.modalH = modalInteriorHeight(m.height, len(lines)+modalChromeLines)
	m.summaryScroll = 0
}

// actWidth returns the delete modal's fixed interior width, deriving a
// fresh one when an act renders without going through the delete key.
func (m Model) actWidth() int {
	if m.modalW > 0 {
		return m.modalW
	}
	return modalInteriorWidth(m.width)
}

// actHeight returns the fixed interior height, sized to the given content
// when no confirm act pinned the frame beforehand.
func (m Model) actHeight(contentLines int) int {
	if m.modalH > 0 {
		return m.modalH
	}
	return modalInteriorHeight(m.height, contentLines)
}

// modalChromeLines is the confirm modal's fixed header (title, hint, blank)
// plus footer (blank, hints).
const modalChromeLines = 5

// padLine pads a (possibly styled) line with spaces to the given width.
func padLine(s string, width int) string {
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// confirmLines renders every plan item into its display lines and records
// where each item starts, for cursor-following scroll.
func (m Model) confirmLines(width int) (lines []string, itemStarts []int) {
	for i, it := range m.confirmItems {
		itemStarts = append(itemStarts, len(lines))
		cursor, cursorStyle := "  ", styleDim
		if i == m.confirmCursor {
			cursor, cursorStyle = "▸ ", styleAccentBold
		}
		branchMark := "[x] delete branch"
		switch {
		case it.BranchOnly():
			branchMark = "(no worktree)"
		case it.Worktree.Detached():
			branchMark = "(detached: no branch)"
		case !it.DeleteBranch:
			branchMark = "[ ] keep branch"
		}
		label := it.Worktree.Branch
		if it.BranchOnly() {
			label = it.Branch
		} else if it.Worktree.Detached() {
			label = leafLabel(it)
		}
		nameStyle := lipgloss.NewStyle()
		if i == m.confirmCursor {
			nameStyle = nameStyle.Bold(true)
		}
		avail := width - 2 - lipgloss.Width(branchMark) - 2
		lines = append(lines,
			cursorStyle.Render(cursor)+nameStyle.Render(padLine(truncate(label, max(avail, 1)), max(avail, 1)))+"  "+styleDim.Render(branchMark))
		if !it.BranchOnly() {
			lines = append(lines, styleDim.Render(truncate("    "+it.Worktree.Path, width)))
		}
		for _, warn := range m.warningsFor(it) {
			lines = append(lines, styleWarn.Render(truncate("    "+warn, width)))
		}
	}
	return lines, itemStarts
}

// viewConfirm renders the batch confirmation act of the delete modal:
// danger title, scrolling item list, and key hints.
func (m Model) viewConfirm() string {
	w := m.actWidth()
	lines, starts := m.confirmLines(w)
	h := m.actHeight(len(lines) + modalChromeLines)
	bodyH := max(h-modalChromeLines, 1)

	// Derive the scroll offset from the cursor so its item is fully visible.
	offset := 0
	if len(starts) > 0 {
		itemStart := starts[min(m.confirmCursor, len(starts)-1)]
		itemEnd := len(lines)
		if m.confirmCursor+1 < len(starts) {
			itemEnd = starts[m.confirmCursor+1]
		}
		if itemEnd > bodyH {
			offset = itemEnd - bodyH
		}
		offset = min(offset, itemStart)
	}

	unit := "worktree(s)"
	if m.branchMode {
		unit = "branch(es)"
	}
	title := styleDanger.Render(fmt.Sprintf("Delete %d %s?", len(m.confirmItems), unit))
	pos := ""
	if len(lines) > bodyH {
		pos = styleDim.Render(fmt.Sprintf("%d/%d", m.confirmCursor+1, len(m.confirmItems)))
	}
	sub := styleDim.Render("space: toggle branch deletion · j/k move")
	sep := styleDim.Render("  ·  ")
	hints := hint("y", "confirm") + sep + hint("n/esc", "cancel")

	out := []string{padLine(title, w-lipgloss.Width(pos)) + pos, sub, ""}
	out = append(out, windowLines(lines, offset, bodyH)...)
	out = append(out, "", hints)
	for i := range out {
		out[i] = padLine(out[i], w)
	}
	return strings.Join(out, "\n")
}

// viewWaiting fills the act frame while deletion results are pending.
func (m Model) viewWaiting() string {
	w, h := m.actWidth(), m.actHeight(modalChromeLines+1)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, styleDim.Render("deleting..."))
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
// never block). Branch-only items warn from the branch's own state.
func (m Model) warningsFor(it action.PlanItem) []string {
	var warns []string
	if it.BranchOnly() {
		if repo := repoByPath(m.repos, it.RepoPath); repo != nil {
			if b := branchByName(repo, it.Branch); b != nil {
				if !b.Merged && !b.UpstreamGone {
					warns = append(warns, "⚠ unmerged (will force-delete)")
				}
				if b.UnpushedCount > 0 {
					warns = append(warns, fmt.Sprintf("⚠ %d commit(s) not pushed to the remote", b.UnpushedCount))
				}
			}
		}
		return warns
	}
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
