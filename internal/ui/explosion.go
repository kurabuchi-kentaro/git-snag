package ui

import (
	"math"
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// The explosion animation is adapted from lazygit's "Nuke working tree"
// effect (pkg/gui/controllers/workspace_reset_controller.go, Explode /
// getExplodeImage by Jesse Duffield and contributors, MIT licensed):
// a radial burst of random symbols with a trailing inner shockwave and a
// square-root progress easing. lazygit drives it with a sleeping goroutine;
// here it is re-expressed as a tea.Tick message loop, and its intensity
// scales with the number of worktrees being deleted. Thanks, lazygit!
// See docs/adr/0007 for the full attribution rationale.

// explosionTier scales the effect to the batch size.
type explosionTier struct {
	frames    int
	radiusMul float64
	shake     bool
	chars     string
}

// tierFor picks the animation intensity: a lone worktree pops quietly, a
// handful matches lazygit's baseline, five or more goes nuclear.
func tierFor(count int) explosionTier {
	switch {
	case count <= 1:
		return explosionTier{frames: 15, radiusMul: 0.85, chars: "*.@#"}
	case count <= 4:
		return explosionTier{frames: 25, radiusMul: 1.0, chars: "*.@#&+%"}
	default:
		return explosionTier{frames: 35, radiusMul: 1.35, shake: true, chars: "*.@#&+%$!"}
	}
}

// explosionFrameInterval matches lazygit's 20ms cadence.
const explosionFrameInterval = 20 * time.Millisecond

// explosionTickMsg advances the animation by one frame.
type explosionTickMsg struct{}

// explosionTick arms the next frame.
func explosionTick() tea.Cmd {
	return tea.Tick(explosionFrameInterval, func(time.Time) tea.Msg {
		return explosionTickMsg{}
	})
}

// renderExplodeFrame draws one frame: symbols fill the ring between the
// expanding outer radius and the shockwave's inner radius, thinning out as
// the explosion progresses. Deterministic for a given seed and frame.
func renderExplodeFrame(width, height, frame int, tier explosionTier, seed int64) string {
	// #nosec G404 -- visual effect only; determinism matters, secrecy does not.
	rng := rand.New(rand.NewSource(seed + int64(frame)))
	centerX, centerY := width/2, height/2
	maxRadius := math.Hypot(float64(centerX), float64(centerY)) * tier.radiusMul
	progress := math.Sqrt(float64(frame) / float64(tier.frames))
	radius := progress * maxRadius * 2
	var innerRadius float64
	if progress > 0.5 {
		innerRadius = (progress - 0.5) * 2 * maxRadius
	}

	chars := []rune(tier.chars)
	var b strings.Builder
	for y := range height {
		for x := range width {
			// Scale y by 2 to compensate for the terminal cell aspect ratio.
			distance := math.Hypot(float64(x-centerX), float64(y-centerY)*2)
			if distance <= radius && distance >= innerRadius && rng.Float64() > progress {
				b.WriteRune(chars[rng.Intn(len(chars))])
			} else {
				b.WriteRune(' ')
			}
		}
		if y < height-1 {
			b.WriteRune('\n')
		}
	}
	return b.String()
}

// explosionStyles cycles white → yellow → red as the burst progresses,
// echoing lazygit's palette within the ANSI-based scheme of ADR 0013.
var explosionStyles = []lipgloss.Style{
	lipgloss.NewStyle(),
	styleWarn,
	styleBad,
}

// viewExplosion renders the current animation frame inside the delete
// modal's act frame (ADR 0013: the blast stays inside the window,
// lazygit-style, instead of taking over the screen).
func (m Model) viewExplosion() string {
	width, height := m.actWidth(), m.actHeight(modalChromeLines+1)
	img := renderExplodeFrame(width, height, m.explosionFrame, m.explosionTier, m.explosionSeed)
	style := explosionStyles[m.explosionFrame*len(explosionStyles)/(m.explosionTier.frames+1)%len(explosionStyles)]
	return style.Render(img)
}

// explosionJitter shakes the modal frame by a cell or two on the big tier.
func (m Model) explosionJitter() (dx, dy int) {
	if !m.explosionTier.shake {
		return 0, 0
	}
	// #nosec G404 -- visual jitter only.
	rng := rand.New(rand.NewSource(m.explosionSeed - int64(m.explosionFrame)))
	return rng.Intn(3) - 1, rng.Intn(2)
}
