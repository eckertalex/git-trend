package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// placeOverlay composites over centered on base, preserving content on both sides.
func placeOverlay(base, over string, w, h int) string {
	baseLines := strings.Split(base, "\n")
	overLines := strings.Split(over, "\n")

	overH := len(overLines)
	overW := 0
	for _, l := range overLines {
		if lw := lipgloss.Width(l); lw > overW {
			overW = lw
		}
	}

	startX := max(0, (w-overW)/2)
	startY := max(0, (h-overH)/2)

	for len(baseLines) < h {
		baseLines = append(baseLines, "")
	}

	for i, overLine := range overLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}
		bl := baseLines[y]
		if lipgloss.Width(bl) < w {
			bl += strings.Repeat(" ", w-lipgloss.Width(bl))
		}
		// Truncate the left segment and reset, so open ANSI codes from the base
		// do not bleed into the overlay content.
		left := ansi.Truncate(bl, startX, "") + "\x1b[0m"
		right := ansi.TruncateLeft(bl, startX+overW, "")
		baseLines[y] = left + overLine + right
	}

	return strings.Join(baseLines, "\n")
}
