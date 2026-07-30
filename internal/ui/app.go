package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

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

	repos           []domain.Repo
	rows            []uiRow
	focus           int
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
			m.phase = phaseBrowsing
			m.results = nil
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

// moveFocus shifts the focused row, clamped at both ends.
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
	if m.showHelp {
		return tea.NewView(m.viewHelp())
	}
	switch m.phase {
	case phaseConfirming:
		return tea.NewView(m.viewConfirm())
	case phaseExploding:
		if m.explosionDone {
			return tea.NewView("deleting..." + "\n")
		}
		return tea.NewView(m.viewExplosion())
	case phaseExecuting:
		return tea.NewView("deleting..." + "\n")
	case phaseSummary:
		return tea.NewView(m.viewSummary())
	case phaseBrowsing:
	}

	var b strings.Builder
	now := m.now()

	b.WriteString(m.statusLine())
	b.WriteString("\n\n")

	if len(m.rows) == 0 {
		if m.scanning {
			b.WriteString(styleDim.Render("scanning..."))
		} else {
			b.WriteString(styleDim.Render("No worktrees match the current view"))
		}
		return tea.NewView(b.String())
	}

	for i, row := range m.rows {
		focused := i == m.focus
		switch row.kind {
		case rowRepo:
			repo := &m.repos[row.repoIdx]
			caret := "▾"
			if m.collapsedRepos[row.key] {
				caret = "▸"
			}
			mark := m.selectionMark(row)
			line := fmt.Sprintf("%s %s %s  %s", caret, mark, styleRepoName.Render(repo.Name), styleDim.Render(repo.Path))
			count := fmt.Sprintf("%d worktrees", m.visibleWorktreeCount(repo))
			line = padBetween(line, styleDim.Render(count), m.width)
			if focused {
				line = styleFocused.Render(fmt.Sprintf("%s %s %s  %s", caret, mark, repo.Name, repo.Path))
			}
			b.WriteString(line + "\n")
		case rowGroup:
			caret := "▾"
			if m.collapsedGroups[row.key] {
				caret = "▸"
			}
			line := connectorPrefix(row.isLast) + caret + " " + m.selectionMark(row) + " " + row.node.Name + "/"
			if focused {
				line = styleFocused.Render(line)
			}
			b.WriteString(line + "\n")
		case rowLeaf:
			prefix := ""
			if !row.flat {
				prefix = connectorPrefix(row.isLast)
			}
			selected := m.selection[row.node.Worktree.Path]
			b.WriteString(renderLeafLine(prefix, *row.node.Worktree, row.node.Name, m.width, now, focused, selected) + "\n")
			pathIndent := strings.Repeat(" ", len(prefix))
			b.WriteString(renderPathLine(pathIndent+"  ", *row.node.Worktree, m.width) + "\n")
		}
	}

	b.WriteString("\n")
	if m.filtering || m.filter != "" {
		b.WriteString(m.filterInput.View() + "\n")
	}
	b.WriteString(m.footer())
	return tea.NewView(b.String())
}

// selectionMark renders the tri-state indicator of a group or repo row.
func (m Model) selectionMark(row uiRow) string {
	switch m.selectionState(row) {
	case selAll:
		return "✓"
	case selPartial:
		return "◐"
	case selNone:
		return "·"
	default:
		return "·"
	}
}

// statusLine summarizes the scan: repo/worktree counts, scan state, and the
// current selection.
func (m Model) statusLine() string {
	total := 0
	for i := range m.repos {
		total += len(m.repos[i].Worktrees)
	}
	s := fmt.Sprintf("%d repos · %d worktrees", len(m.repos), total)
	if m.scanning {
		s += " · scanning..."
	}
	if n := len(m.selection); n > 0 {
		s += fmt.Sprintf(" · %d selected", n)
	}
	if m.visualAnchor >= 0 {
		s += " · VISUAL"
	}
	if m.mergedOnly {
		s += " · merged only"
	}
	if m.sortMode.Flat() {
		s += " · " + sortModeLabel(m.sortMode)
	}
	return styleDim.Render(s)
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

// footer renders the key hints.
func (m Model) footer() string {
	hints := []string{
		"j/k move", "space select", "v visual", "d delete",
		"h/l collapse", "/ filter", "m merged only", "s sort", "q quit",
	}
	return styleDim.Render(strings.Join(hints, "  ·  "))
}
