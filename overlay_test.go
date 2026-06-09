package main

import (
	"strings"
	"testing"
)

func TestPlaceOverlayCenter(t *testing.T) {
	// 20-wide base, 5-wide overlay — should be centered horizontally.
	base := strings.Repeat("abcdefghijklmnopqrst\n", 5)
	over := "XXXXX\nYYYYY\nZZZZZ"

	result := placeOverlay(base, over, 20, 5)
	lines := strings.Split(result, "\n")

	// Overlay starts at col (20-5)/2 = 7.
	for i, want := range []string{"XXXXX", "YYYYY", "ZZZZZ"} {
		line := lines[1+i] // startY = (5-3)/2 = 1
		if !strings.Contains(line, want) {
			t.Errorf("line %d = %q, want it to contain %q", 1+i, line, want)
		}
	}
}

func TestPlaceOverlayPreservesBaseWidth(t *testing.T) {
	// Ensure lines not covered by the overlay are unchanged.
	base := strings.Repeat("12345678901234567890\n", 5)
	over := "AAA"

	result := placeOverlay(base, over, 20, 5)
	lines := strings.Split(result, "\n")

	// Line 0 and line 4 are outside the overlay (startY = 2 for a 1-line overlay).
	if !strings.HasPrefix(lines[0], "1234567890") {
		t.Errorf("line 0 unexpectedly modified: %q", lines[0])
	}
}

func TestPlaceOverlayWithANSIBase(t *testing.T) {
	// Base line with ANSI color codes; overlay must not corrupt visible positions.
	red := "\x1b[31m"
	reset := "\x1b[0m"
	coloredLine := red + "hello world" + reset + "        plain"
	base := coloredLine + "\n" + coloredLine + "\n" + coloredLine

	over := "OVR"
	result := placeOverlay(base, over, 20, 3)

	// The overlay text must appear somewhere in the result.
	if !strings.Contains(result, "OVR") {
		t.Errorf("overlay text missing from result:\n%s", result)
	}
}

func TestPlaceOverlaySmallBase(t *testing.T) {
	// Base smaller than overlay — should not panic.
	result := placeOverlay("ab\ncd", "OVERLAY", 2, 2)
	_ = result // just confirm it doesn't panic
}
