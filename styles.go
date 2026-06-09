package main

import "github.com/charmbracelet/lipgloss"

var (
	colorFocus  = lipgloss.AdaptiveColor{Light: "#2563eb", Dark: "#60a5fa"} // blue-600 / blue-400
	colorBorder = lipgloss.AdaptiveColor{Light: "#94a3b8", Dark: "#475569"} // slate-400 / slate-600
	colorError  = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#f87171"} // red-600 / red-400
	colorWarn   = lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#fbbf24"} // amber-600 / amber-400
	colorActive = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34d399"} // emerald-600 / emerald-400
	selectedBg  = colorFocus
	selectedFg  = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"}
)

var (
	borderNormal  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorBorder)
	borderFocused = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorFocus)
	styleActive   = lipgloss.NewStyle().Foreground(colorActive)
	styleDim      = lipgloss.NewStyle().Faint(true)
	styleError    = lipgloss.NewStyle().Foreground(colorError)
	styleBold     = lipgloss.NewStyle().Bold(true)
)

var palette = []lipgloss.AdaptiveColor{
	{Light: "#0891b2", Dark: "#22d3ee"}, // cyan
	{Light: "#d97706", Dark: "#fbbf24"}, // amber
	{Light: "#c026d3", Dark: "#e879f9"}, // fuchsia
	{Light: "#059669", Dark: "#34d399"}, // emerald
	{Light: "#7c3aed", Dark: "#a78bfa"}, // violet
	{Light: "#2563eb", Dark: "#60a5fa"}, // blue
	{Light: "#65a30d", Dark: "#a3e635"}, // lime
	{Light: "#dc2626", Dark: "#f87171"}, // red
}

var maxSeries = len(palette)
