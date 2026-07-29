package ui

import (
	"strings"
	"testing"

	"github.com/kurabuchi-kentaro/git-snag/internal/action"
	"github.com/kurabuchi-kentaro/git-snag/internal/domain"
)

func TestTierFor_scalesWithItemCount(t *testing.T) {
	t.Parallel()
	one, few, many := tierFor(1), tierFor(3), tierFor(6)
	if one.frames >= few.frames || few.frames >= many.frames {
		t.Errorf("frames should grow with count: %d, %d, %d", one.frames, few.frames, many.frames)
	}
	if one.shake || few.shake {
		t.Error("small tiers should not shake")
	}
	if !many.shake {
		t.Error("the big tier should shake")
	}
}

func TestRenderExplodeFrame_sizeAndCharset(t *testing.T) {
	t.Parallel()
	tier := tierFor(3)
	const width, height = 40, 10
	for _, frame := range []int{0, tier.frames / 2, tier.frames - 1} {
		img := renderExplodeFrame(width, height, frame, tier, 42)
		lines := strings.Split(img, "\n")
		if len(lines) != height {
			t.Fatalf("frame %d: %d lines, want %d", frame, len(lines), height)
		}
		for _, line := range lines {
			if got := len([]rune(line)); got != width {
				t.Errorf("frame %d: line width %d, want %d", frame, got, width)
			}
			for _, r := range line {
				if r != ' ' && !strings.ContainsRune(tier.chars, r) {
					t.Errorf("frame %d: unexpected rune %q", frame, r)
				}
			}
		}
	}
}

func TestRenderExplodeFrame_deterministicPerSeed(t *testing.T) {
	t.Parallel()
	tier := tierFor(2)
	a := renderExplodeFrame(30, 8, 5, tier, 7)
	b := renderExplodeFrame(30, 8, 5, tier, 7)
	if a != b {
		t.Error("same seed and frame should render identically")
	}
}

// enter the exploding phase with one selected leaf; the Execute cmd is
// produced but never run, so tests inject results manually.
func exploding(t *testing.T) Model {
	t.Helper()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space())
	next, _ := m.Update(press('d'))
	m = next.(Model)
	next, cmd := m.Update(press('y'))
	m = next.(Model)
	if cmd == nil {
		t.Fatal("y should produce commands")
	}
	if m.phase != phaseExploding {
		t.Fatalf("phase = %v, want exploding", m.phase)
	}
	return m
}

func results() DeleteResultsMsg {
	return DeleteResultsMsg{Results: []action.Result{{
		Item: action.PlanItem{Worktree: domain.Worktree{Path: "/work/alpha-wt/feature/a", Branch: "feature/a"}},
	}}}
}

func TestExplosion_animationEndThenResultsReachesSummary(t *testing.T) {
	t.Parallel()
	m := exploding(t)
	for range m.explosionTier.frames + 1 {
		m = apply(t, m, explosionTickMsg{})
	}
	view := m.View().Content
	if !strings.Contains(view, "deleting") {
		t.Errorf("waiting view expected while results pending:\n%s", view)
	}
	m = apply(t, m, results())
	if m.phase != phaseSummary {
		t.Errorf("phase = %v, want summary once both animation and results are done", m.phase)
	}
}

func TestExplosion_resultsDuringAnimationWaitForFrames(t *testing.T) {
	t.Parallel()
	m := exploding(t)
	m = apply(t, m, results())
	if m.phase != phaseExploding {
		t.Fatalf("phase = %v, want still exploding until animation ends", m.phase)
	}
	for range m.explosionTier.frames + 1 {
		m = apply(t, m, explosionTickMsg{})
	}
	if m.phase != phaseSummary {
		t.Errorf("phase = %v, want summary after final frame", m.phase)
	}
}

func TestExplosion_anyKeySkipsAnimation(t *testing.T) {
	t.Parallel()
	m := exploding(t)
	m = apply(t, m, results())
	m = apply(t, m, press('x'))
	if m.phase != phaseSummary {
		t.Errorf("phase = %v, want summary right after keypress skip", m.phase)
	}
}

func TestExplosion_disabledGoesStraightToExecuting(t *testing.T) {
	t.Parallel()
	m := modelWith(t, testRepo("alpha", "feature/a"))
	m.animationEnabled = false
	focusOn(t, &m, leafNamed("a"))
	m = apply(t, m, space(), press('d'))
	next, cmd := m.Update(press('y'))
	m = next.(Model)
	if m.phase != phaseExecuting {
		t.Fatalf("phase = %v, want executing without animation", m.phase)
	}
	if cmd == nil {
		t.Fatal("execute command still expected")
	}
	m = apply(t, m, results())
	if m.phase != phaseSummary {
		t.Errorf("phase = %v, want summary", m.phase)
	}
}
