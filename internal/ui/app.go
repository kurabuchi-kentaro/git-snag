package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kurabuchi-kentaro/git-snag/internal/action"
	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

// Key strings shared between the raw handlers (confirm screen, filter
// input) that bypass the bindings table.
const (
	keyUp   = "up"
	keyDown = "down"
)

// phase is the top-level UI state machine.
type phase int

const (
	phaseBrowsing phase = iota
	phaseConfirming
	phaseExploding
	phaseExecuting
	phaseSummary
)

// Model is the root Bubble Tea model of the browsing UI.
type Model struct {
	phase            phase
	confirmItems     []action.PlanItem
	confirmCursor    int
	results          []action.Result
	pendingResults   []action.Result
	resultsArrived   bool
	animationEnabled bool
	explosionFrame   int
	explosionDone    bool
	explosionTier    explosionTier
	explosionSeed    int64
	// The delete flow renders as three acts in one modal (ADR 0013); the
	// frame's interior size is fixed when the confirm opens.
	modalW        int
	modalH        int
	summaryScroll int
	icons         iconSet

	repos           []domain.Repo
	rows            []uiRow
	focus           int
	scrollLine      int
	collapsedRepos  map[string]bool
	collapsedGroups map[string]bool
	sortMode        domain.SortMode
	mergedOnly      bool
	filter          string
	filterInput     textinput.Model
	filtering       bool
	scanning        bool
	showHelp        bool
	selection       map[string]bool
	visualAnchor    int
	preVisual       map[string]bool
	width           int
	height          int
	now             func() time.Time
}

// NewModel returns an empty model waiting for RepoFoundMsg streams.
func NewModel() Model {
	ti := textinput.New()
	ti.Placeholder = "filter by branch or path"
	ti.Prompt = "/ "
	// The filter prompt is an operation, so it carries the accent (ADR 0013).
	styles := ti.Styles()
	styles.Focused.Prompt = styleAccentBold
	styles.Blurred.Prompt = styleAccent
	ti.SetStyles(styles)
	return Model{
		collapsedRepos:   map[string]bool{},
		collapsedGroups:  map[string]bool{},
		filterInput:      ti,
		scanning:         true,
		selection:        map[string]bool{},
		visualAnchor:     -1,
		animationEnabled: true,
		width:            80,
		height:           24,
		now:              time.Now,
		icons:            iconsNerd,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RepoFoundMsg:
		m.repos = append(m.repos, msg.Repo)
		sort.Slice(m.repos, func(i, j int) bool { return m.repos[i].Path < m.repos[j].Path })
		m.rebuildRows()
		return m, nil

	case ScanDoneMsg:
		m.scanning = false
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ensureFocusVisible()
		// The delete-flow modal keeps its size across acts but must never
		// outgrow a shrunken terminal.
		m.modalW = min(m.modalW, modalInteriorWidth(m.width))
		m.modalH = min(m.modalH, m.height-6)
		return m, nil

	case DeleteResultsMsg:
		if m.phase == phaseExploding && !m.explosionDone {
			// Animation still playing: hold the results (REQ-P5).
			m.pendingResults = msg.Results
			m.resultsArrived = true
			return m, nil
		}
		m.applyResults(msg)
		return m, nil

	case explosionTickMsg:
		if m.phase != phaseExploding || m.explosionDone {
			return m, nil
		}
		m.explosionFrame++
		if m.explosionFrame >= m.explosionTier.frames {
			return m.finishExplosion()
		}
		return m, explosionTick()

	case tea.MouseClickMsg:
		if m.visualAnchor >= 0 {
			m.confirmVisual()
		}
		return m, nil

	case tea.KeyPressMsg:
		switch m.phase {
		case phaseConfirming:
			return m.updateConfirmKey(msg)
		case phaseExploding:
			// Any key skips the animation instantly.
			return m.finishExplosion()
		case phaseExecuting:
			return m, nil
		case phaseSummary:
			// j/k scroll a long summary; anything else closes it.
			switch msg.String() {
			case "j", keyDown:
				w := m.actWidth()
				lines := len(m.summaryLines(w))
				h := m.actHeight(lines + modalChromeLines)
				if maxOff := lines - (h - modalChromeLines); m.summaryScroll < maxOff {
					m.summaryScroll++
				}
				return m, nil
			case "k", keyUp:
				if m.summaryScroll > 0 {
					m.summaryScroll--
				}
				return m, nil
			}
			m.phase = phaseBrowsing
			m.results = nil
			m.summaryScroll = 0
			return m, nil
		case phaseBrowsing:
		}
		return m.updateKey(msg)
	}
	return m, nil
}

// finishExplosion ends the animation; the summary appears once the results
// are also in, otherwise a waiting state holds until they arrive.
func (m Model) finishExplosion() (tea.Model, tea.Cmd) {
	m.explosionDone = true
	if m.resultsArrived {
		m.applyResults(DeleteResultsMsg{Results: m.pendingResults})
		m.pendingResults = nil
		m.resultsArrived = false
	}
	return m, nil
}

// updateKey routes key presses. The filter input takes priority when
// focused; visual mode accepts only its own keys (REQ-A5).
func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	if m.filtering {
		switch msg.String() {
		case "esc", "enter":
			m.filtering = false
			m.filterInput.Blur()
			return m, nil
		case keyUp, keyDown:
			// fzf-style: arrows move the focus without leaving the input
			// (j/k stay literal — branch names contain them).
			if msg.String() == keyDown {
				m.moveFocus(1)
			} else {
				m.moveFocus(-1)
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)
			m.filter = m.filterInput.Value()
			m.rebuildRows()
			return m, cmd
		}
	}

	if m.visualAnchor >= 0 {
		switch {
		case key.Matches(msg, keys.Down):
			m.moveFocus(1)
			m.applyVisualRange()
		case key.Matches(msg, keys.Up):
			m.moveFocus(-1)
			m.applyVisualRange()
		case key.Matches(msg, keys.Visual):
			m.confirmVisual()
		case key.Matches(msg, keys.Escape):
			m.revertVisual()
		}
		// Everything else — /, m, s, d, even q — is ignored in visual mode.
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Down):
		m.moveFocus(1)
	case key.Matches(msg, keys.Up):
		m.moveFocus(-1)
	case key.Matches(msg, keys.Collapse):
		m.collapseFocused()
	case key.Matches(msg, keys.Expand):
		m.expandFocused()
	case key.Matches(msg, keys.Select):
		if len(m.rows) > 0 {
			m.toggleSelection(m.rows[m.focus])
		}
	case key.Matches(msg, keys.Visual):
		if len(m.rows) > 0 {
			m.enterVisual()
			m.applyVisualRange()
		}
	case key.Matches(msg, keys.Escape):
		m.selection = map[string]bool{}
	case key.Matches(msg, keys.Delete):
		if len(m.selection) > 0 {
			m.confirmItems = m.buildPlan()
			m.confirmCursor = 0
			m.phase = phaseConfirming
			m.sizeModal()
		}
	case key.Matches(msg, keys.Filter):
		m.filtering = true
		return m, m.filterInput.Focus()
	case key.Matches(msg, keys.Merged):
		m.mergedOnly = !m.mergedOnly
		m.rebuildRows()
	case key.Matches(msg, keys.Sort):
		m.sortMode = m.sortMode.Next()
		m.rebuildRows()
	case key.Matches(msg, keys.Help):
		m.showHelp = true
	}
	return m, nil
}

// moveFocus shifts the focused row, clamped at both ends, keeping the
// focused row inside the visible window.
func (m *Model) moveFocus(delta int) {
	m.focus += delta
	if m.focus < 0 {
		m.focus = 0
	}
	if m.focus >= len(m.rows) {
		m.focus = len(m.rows) - 1
	}
	if m.focus < 0 {
		m.focus = 0
	}
	m.ensureFocusVisible()
}

// rowHeight is the number of terminal lines one row occupies.
func rowHeight(r uiRow) int {
	if r.kind == rowLeaf {
		return 2 // branch line + path line
	}
	return 1
}

// contentHeight is the number of lines available to the tree pane after the
// status line, footer, and optional filter input take their share.
func (m *Model) contentHeight() int {
	chrome := 4 // status + blank above, blank + footer below
	if m.filtering || m.filter != "" {
		chrome++
	}
	h := m.height - chrome
	if h < 1 {
		return 1
	}
	return h
}

// ensureFocusVisible scrolls the tree pane so the focused row is fully on
// screen. Without this, rows past the terminal height would be rendered but
// clipped — the focus would move invisibly.
func (m *Model) ensureFocusVisible() {
	if len(m.rows) == 0 {
		m.scrollLine = 0
		return
	}
	total := 0
	start := 0
	for i, r := range m.rows {
		if i == m.focus {
			start = total
		}
		total += rowHeight(r)
	}
	end := start + rowHeight(m.rows[m.focus])
	contentH := m.contentHeight()

	if maxScroll := total - contentH; m.scrollLine > maxScroll {
		m.scrollLine = maxScroll
	}
	if m.scrollLine < 0 {
		m.scrollLine = 0
	}
	if start < m.scrollLine {
		m.scrollLine = start
	}
	if end > m.scrollLine+contentH {
		m.scrollLine = end - contentH
	}
}

// collapseFocused folds the focused group/repo, or — on a leaf — its nearest
// ancestor, moving focus there.
func (m *Model) collapseFocused() {
	if len(m.rows) == 0 {
		return
	}
	row := m.rows[m.focus]
	switch row.kind {
	case rowRepo:
		m.collapsedRepos[row.key] = true
	case rowGroup:
		m.collapsedGroups[row.key] = true
	case rowLeaf:
		target := row.parentKey
		if _, _, isGroup := splitGroupKey(target); isGroup {
			m.collapsedGroups[target] = true
		} else {
			m.collapsedRepos[target] = true
		}
		m.rebuildRows()
		for i, r := range m.rows {
			if r.key == target {
				m.focus = i
				break
			}
		}
		m.ensureFocusVisible()
		return
	}
	m.rebuildRows()
}

// expandFocused unfolds the focused group or repo header.
func (m *Model) expandFocused() {
	if len(m.rows) == 0 {
		return
	}
	row := m.rows[m.focus]
	switch row.kind {
	case rowRepo:
		delete(m.collapsedRepos, row.key)
	case rowGroup:
		delete(m.collapsedGroups, row.key)
	case rowLeaf:
		return
	}
	m.rebuildRows()
}

// View implements tea.Model.
func (m Model) View() tea.View {
	content := m.viewBrowse()
	if m.showHelp {
		content = overlayModal(content, m.viewHelp(), m.width, m.height, 0, 0)
	} else {
		switch m.phase {
		case phaseConfirming:
			content = overlayModal(content, m.viewConfirm(), m.width, m.height, 0, 0)
		case phaseExploding:
			if m.explosionDone {
				content = overlayModal(content, m.viewWaiting(), m.width, m.height, 0, 0)
			} else {
				dx, dy := m.explosionJitter()
				content = overlayModal(content, m.viewExplosion(), m.width, m.height, dx, dy)
			}
		case phaseExecuting:
			content = overlayModal(content, m.viewWaiting(), m.width, m.height, 0, 0)
		case phaseSummary:
			content = overlayModal(content, m.viewSummary(), m.width, m.height, 0, 0)
		case phaseBrowsing:
		}
	}
	// Full-window mode, lazygit-style: the tool owns the screen while it
	// runs and restores the shell on exit.
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// viewBrowse renders the tree pane with its status line and footer; it is
// also the backdrop every modal composites over.
func (m Model) viewBrowse() string {
	var b strings.Builder
	now := m.now()
	rule := styleDim.Render(strings.Repeat("─", max(m.width, 1)))

	b.WriteString(m.statusLine())
	b.WriteString("\n" + rule + "\n")

	if len(m.rows) == 0 {
		if m.scanning {
			b.WriteString(styleDim.Render("scanning..."))
		} else {
			b.WriteString(styleDim.Render("No worktrees match the current view"))
		}
		return b.String()
	}

	// Render only the rows inside the scroll window: this both keeps the
	// output within the terminal height (rows past it would be clipped
	// invisibly) and bounds per-frame work by the screen size instead of
	// the total worktree count.
	contentH := m.contentHeight()
	lineIdx, written := 0, 0
	for i, row := range m.rows {
		h := rowHeight(row)
		if lineIdx+h <= m.scrollLine {
			lineIdx += h
			continue
		}
		if written >= contentH {
			break
		}
		for _, line := range m.renderRow(i, row, now) {
			if lineIdx >= m.scrollLine && written < contentH {
				b.WriteString(line + "\n")
				written++
			}
			lineIdx++
		}
	}
	// Pad the pane so the footer stays pinned to the bottom of the screen.
	for ; written < contentH; written++ {
		b.WriteString("\n")
	}

	b.WriteString(rule + "\n")
	if m.filtering || m.filter != "" {
		b.WriteString(m.filterInput.View() + "\n")
	}
	b.WriteString(m.footer())
	return b.String()
}

// rowStateFor derives the display states of row i, which may all hold at
// once: focus, selection, and the visual-mode range.
func (m Model) rowStateFor(i int, selected bool) rowState {
	st := rowState{focused: i == m.focus, selected: selected}
	if m.visualAnchor >= 0 {
		lo, hi := m.visualAnchor, m.focus
		if lo > hi {
			lo, hi = hi, lo
		}
		st.inRange = i >= lo && i <= hi
	}
	return st
}

// renderRow renders one row into its terminal lines (1 for headers and
// groups, 2 for leaves).
func (m Model) renderRow(i int, row uiRow, now time.Time) []string {
	switch row.kind {
	case rowRepo:
		st := m.rowStateFor(i, false)
		repo := &m.repos[row.repoIdx]
		caret := "▾"
		if m.collapsedRepos[row.key] {
			caret = "▸"
		}
		mark := m.selectionMark(row)
		count := fmt.Sprintf("%d worktrees", m.visibleWorktreeCount(repo))
		width := m.width - 1 // gutter
		fixed := 2 + lipgloss.Width(mark) + lipgloss.Width(repo.Name) + 2
		pathAvail := width - fixed - lipgloss.Width(count) - 1
		path := truncate(repo.Path, max(pathAvail, 1))
		pad := max(width-fixed-lipgloss.Width(path)-lipgloss.Width(count), 1)

		var b strings.Builder
		b.WriteString(gutter(st))
		b.WriteString(bgIf(styleDim, st.focused).Render(caret + " "))
		if mark != "" {
			b.WriteString(bgIf(styleSelected, st.focused).Render(mark))
		}
		b.WriteString(bgIf(styleRepoName, st.focused).Render(repo.Name))
		b.WriteString(bgIf(styleDim, st.focused).Render("  " + path))
		b.WriteString(bgIf(lipgloss.NewStyle(), st.focused).Render(strings.Repeat(" ", pad)))
		b.WriteString(bgIf(styleDim, st.focused).Render(count))
		return []string{b.String()}
	case rowGroup:
		st := m.rowStateFor(i, false)
		caret := "▾"
		if m.collapsedGroups[row.key] {
			caret = "▸"
		}
		mark := m.selectionMark(row)
		var b strings.Builder
		b.WriteString(gutter(st))
		b.WriteString(bgIf(styleDim, st.focused).Render(connectorPrefix(row.isLast) + caret + " "))
		if mark != "" {
			b.WriteString(bgIf(styleSelected, st.focused).Render(mark))
		}
		b.WriteString(bgIf(styleGroup, st.focused).Render(row.node.Name + "/"))
		return []string{b.String()}
	case rowLeaf:
		prefix, pathPrefix := "", "  "
		if !row.flat {
			prefix = connectorPrefix(row.isLast)
			pathPrefix = pathConnectorPrefix(row.isLast) + "  "
		}
		wt := row.node.Worktree
		st := m.rowStateFor(i, m.selection[wt.Path])
		st.muted = wt.Branch != "" && wt.Branch == m.repos[row.repoIdx].DefaultBranch
		return []string{
			renderLeafLine(prefix, *wt, row.node.Name, m.width, now, st, m.icons),
			renderPathLine(pathPrefix, *wt, m.width, st),
		}
	default:
		return nil
	}
}

// selectionMark returns the tri-state indicator of a group or repo row.
// Unselected rows show nothing: the caret already marks these rows, so a
// placeholder dot would just duplicate it.
func (m Model) selectionMark(row uiRow) string {
	switch m.selectionState(row) {
	case selAll:
		return "✓ "
	case selPartial:
		return "◐ "
	default:
		return ""
	}
}

// statusLine summarizes the scan: repo/worktree counts, scan state, and the
// current selection. Operational states (selection count, visual mode) get
// the purple accent; the rest stays faint.
func (m Model) statusLine() string {
	total := 0
	for i := range m.repos {
		total += len(m.repos[i].Worktrees)
	}
	sep := styleDim.Render(" · ")
	parts := []string{styleDim.Render(fmt.Sprintf("%d repos · %d worktrees", len(m.repos), total))}
	if m.scanning {
		parts = append(parts, styleDim.Render("scanning..."))
	}
	if n := len(m.selection); n > 0 {
		parts = append(parts, styleAccentBold.Render(fmt.Sprintf("%d selected", n)))
	}
	if m.visualAnchor >= 0 {
		parts = append(parts, styleBadge.Render(" VISUAL "))
	}
	if m.mergedOnly {
		parts = append(parts, styleGood.Render("merged only"))
	}
	if m.sortMode.Flat() {
		parts = append(parts, styleDim.Render(sortModeLabel(m.sortMode)))
	}
	return " " + strings.Join(parts, sep)
}

func sortModeLabel(mode domain.SortMode) string {
	switch mode {
	case domain.StaleFirst:
		return "oldest first"
	case domain.FreshFirst:
		return "newest first"
	case domain.MergedFirst:
		return "merged first"
	case domain.TreeView:
		return "tree"
	default:
		return "tree"
	}
}

// hint renders one footer entry: the key bold, its description faint.
func hint(k, desc string) string {
	return styleKey.Render(k) + " " + styleDim.Render(desc)
}

// accentHint renders a footer entry emphasized in the accent color, used to
// steer the user toward the next step of the delete flow.
func accentHint(k, desc string) string {
	return styleAccentBold.Render(k) + " " + styleAccent.Render(desc)
}

// footer renders the key hints. The set swaps with context (ADR 0013):
// browsing, pending selection, and visual mode each show their own next
// steps.
func (m Model) footer() string {
	sep := styleDim.Render("  ·  ")
	var hints []string
	switch {
	case m.visualAnchor >= 0:
		hints = []string{
			styleBadge.Render(" VISUAL "),
			hint("j/k", "extend"), hint("v", "confirm"), hint("esc", "cancel"),
		}
	case len(m.selection) > 0:
		hints = []string{
			styleAccentBold.Render(fmt.Sprintf("%d selected", len(m.selection))),
			accentHint("d", "delete"),
			hint("space", "toggle"), hint("v", "visual"),
			hint("j/k", "move"), hint("esc", "clear"),
		}
	default:
		hints = []string{
			hint("j/k", "move"), hint("space", "select"), hint("v", "visual"),
			hint("h/l", "collapse"), hint("/", "filter"), hint("m", "merged only"),
			hint("s", "sort"), hint("?", "help"), hint("q", "quit"),
		}
	}
	return " " + strings.Join(hints, sep)
}
