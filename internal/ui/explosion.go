package ui

import (
	"math"
	"math/rand"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
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

// viewExplosion renders the current animation frame, jittering the whole
// pane on the big tier.
func (m Model) viewExplosion() string {
	height := m.height - 4
	if height < 5 {
		height = 5
	}
	width := m.width
	if width < 10 {
		width = 10
	}
	img := renderExplodeFrame(width, height, m.explosionFrame, m.explosionTier, m.explosionSeed)
	if m.explosionTier.shake {
		// #nosec G404 -- visual jitter only.
		rng := rand.New(rand.NewSource(m.explosionSeed - int64(m.explosionFrame)))
		img = strings.Repeat("\n", rng.Intn(2)) + strings.Repeat(" ", rng.Intn(3)) + img
	}
	return img
}
