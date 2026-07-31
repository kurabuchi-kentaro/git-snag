package ui

import (
	uv "github.com/charmbracelet/ultraviolet"

	"charm.land/lipgloss/v2"
)

// overlayModal composites a purple-framed modal over the base view: the
// backdrop is dimmed cell by cell, the modal is centered, and dx/dy jitter
// its position (explosion shake). ADR 0013.
//
// The frame is drawn as a uv.StyledString at its own rectangle: Canvas's
// Compose ignores layer positions (that is the Compositor's job) and a
// StyledString clears whatever area it is given, so composing the frame
// over the full canvas would erase the backdrop and pin the modal top-left.
func overlayModal(base, content string, width, height, dx, dy int) string {
	if width < 1 || height < 1 {
		return base
	}
	canvas := lipgloss.NewCanvas(width, height)
	uv.NewStyledString(base).Draw(canvas, canvas.Bounds())
	for y := range height {
		for x := range width {
			if cell := canvas.CellAt(x, y); cell != nil {
				cell.Style.Attrs |= uv.AttrFaint
			}
		}
	}

	frame := styleModal.Render(content)
	fw, fh := lipgloss.Width(frame), lipgloss.Height(frame)
	x := max((width-fw)/2+dx, 0)
	y := max((height-fh)/2+dy, 0)
	uv.NewStyledString(frame).Draw(canvas, uv.Rect(x, y, fw, fh))
	return canvas.Render()
}

// modalInteriorWidth caps the usable content width of a modal: two thirds
// of the terminal, at most 100 columns, never wider than the screen allows.
func modalInteriorWidth(termWidth int) int {
	w := termWidth * 2 / 3
	if w > 100 {
		w = 100
	}
	if w > termWidth-8 {
		w = termWidth - 8
	}
	return max(w, 20)
}

// modalInteriorHeight caps the usable content height of a modal.
func modalInteriorHeight(termHeight, contentLines int) int {
	h := termHeight - 6 // margins + frame border
	if contentLines < h {
		h = contentLines
	}
	return max(h, 3)
}

// windowLines returns the scroll window [offset, offset+height) of lines,
// padding short content so the modal keeps its fixed height across the
// confirm → explosion → summary acts.
func windowLines(lines []string, offset, height int) []string {
	if offset > len(lines)-height {
		offset = len(lines) - height
	}
	offset = max(offset, 0)
	out := make([]string, 0, height)
	for i := offset; i < offset+height; i++ {
		if i < len(lines) {
			out = append(out, lines[i])
		} else {
			out = append(out, "")
		}
	}
	return out
}
