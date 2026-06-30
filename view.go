package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func (m tuiModel) View() string {
	if m.width == 0 {
		return "loading…\n"
	}
	h := m.panelHeight()
	left := m.renderLeft(h)
	right := m.renderChartPanel(h)
	if m.rightView == viewAuthor {
		right = m.renderAuthorPanel(h)
	}
	main := lipgloss.JoinHorizontal(lipgloss.Top, left, right) + "\n" + m.renderFooter()
	if m.mode == modeMenu {
		return placeOverlay(main, m.renderMenu(), m.width, m.height)
	}
	return main
}

func (m tuiModel) renderLeft(h int) string {
	innerW := leftPanelWidth - 2

	// In the author view the profile panel is the active context, so neither left
	// box draws a focused border (the authors list still shows its selection).
	chartFocus := m.rightView == viewChart

	const tfInnerH = 4 // 2-line compact delegate × 2 items, no spacing
	tfBox := m.renderBox("Time range", m.timeRangeList.View(), chartFocus && m.focusedBox == boxTimeFrames, innerW, tfInnerH)

	authInnerH := m.authorsPanelHeight()
	authTitle := fmt.Sprintf("Authors %d/%d", m.usedSeriesCount(), maxSeries)
	authBox := m.renderBox(authTitle, m.authorsList.View(), chartFocus && m.focusedBox == boxAuthors, innerW, authInnerH)

	return lipgloss.NewStyle().Width(leftPanelWidth).Height(h).Render(authBox + "\n" + tfBox)
}

func (m tuiModel) renderBox(title, content string, focused bool, innerW, innerH int) string {
	s := borderNormal
	if focused {
		s = borderFocused
	}
	rendered := s.Width(innerW).Height(innerH).Render(content)
	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		color := colorBorder
		if focused {
			color = colorFocus
		}
		lines[0] = titleBorderLine(title, innerW, focused, color)
	}
	return strings.Join(lines, "\n")
}

// titleBorderLine produces the same visual width in both forms -- no layout
// shift on focus change.
func titleBorderLine(title string, innerW int, focused bool, color lipgloss.AdaptiveColor) string {
	cs := lipgloss.NewStyle().Foreground(color)
	labelW := len([]rune(title)) + 2 // " title " and "[title]" are the same width
	dashes := max(innerW-labelW, 0)
	tail := cs.Render(strings.Repeat("─", dashes) + "╮")
	if focused {
		label := cs.Render("[") +
			lipgloss.NewStyle().Foreground(color).Bold(true).Render(title) +
			cs.Render("]")
		return cs.Render("╭") + label + tail
	}
	return cs.Render("╭ " + title + " " + strings.Repeat("─", dashes) + "╮")
}

func (m tuiModel) renderChartPanel(h int) string {
	w := m.width - leftPanelWidth
	innerW := w - 2
	innerH := h - 2

	var body string
	switch {
	case m.loading && m.chartView == "":
		body = styleDim.Render(m.spinner.View() + " Fetching commits…")
	case m.buildErr != nil:
		body = styleError.Render("error: " + m.buildErr.Error())
	case m.chartView != "":
		body = m.chartView
	case len(m.allCommits) == 0:
		body = styleDim.Render("No commits in the selected range.\n\n[s] since  [u] until  [d] clear")
	default:
		body = styleDim.Render("No active series.\n\n[T] toggle total  [a] add author  [t] fill top")
	}

	rendered := borderNormal.Width(innerW).Height(innerH).Render(body)
	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		title := m.repoName
		if title == "" {
			title = "Chart"
		}
		lines[0] = titleBorderLine(title, innerW, false, colorBorder)
	}
	return strings.Join(lines, "\n")
}

// renderAuthorPanel wraps the scrollable profile viewport in a focused, titled box.
func (m tuiModel) renderAuthorPanel(h int) string {
	innerW := m.chartWidth() // matches the viewport width set in sizeAuthorViewport
	innerH := h - 2

	body := m.authorVP.View()
	if m.detailView == "" {
		body = styleDim.Render("No commits for this author.")
	}

	rendered := borderFocused.Width(innerW).Height(innerH).Render(body)
	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		name, _ := parseNameEmail(m.detailAuthor)
		lines[0] = titleBorderLine(name, innerW, true, colorFocus)
	}
	return strings.Join(lines, "\n")
}

func (m tuiModel) renderFooter() string {
	if m.mode == modeAddAuthor {
		prompt := styleBold.Render("add:") + " " + m.authorInput.View() +
			"  " + styleDim.Render("<enter> confirm  <esc> cancel")
		return lipgloss.NewStyle().Width(m.width).Render(prompt)
	}
	if m.mode == modeEditSince || m.mode == modeEditUntil {
		label := "since"
		if m.mode == modeEditUntil {
			label = "until"
		}
		prompt := styleBold.Render(label+":") + " " + m.dateInput.View() +
			"  " + styleDim.Render("<enter> confirm  <esc> cancel")
		return lipgloss.NewStyle().Width(m.width).Render(prompt)
	}

	if m.statusMsg != "" {
		return lipgloss.NewStyle().Width(m.width).Foreground(colorWarn).Render(m.statusMsg)
	}

	sep := styleDim.Render(" | ")
	var parts []string
	if m.loading && m.chartView != "" {
		parts = append(parts, styleDim.Render(m.spinner.View()+" refreshing…"))
	}
	if m.rightView == viewAuthor {
		parts = append(parts,
			hintKeys("j/k", "switch author"),
			hintKeys("ctrl+u/d", "scroll"),
			hint(keys.Add),
			hint(keys.Delete),
			hintCustom(keys.Esc, "back"),
			hint(keys.Quit),
		)
	} else {
		switch m.focusedBox {
		case boxAuthors:
			parts = append(parts,
				hintCustom(keys.Select, "details"),
				hint(keys.Add),
				hint(keys.Delete),
				hintCustom(keys.Total, "total "+onOff(m.showTotal)),
				hint(keys.Top),
				hint(keys.Web),
				hint(keys.Help),
				hint(keys.Quit),
			)
		case boxTimeFrames:
			parts = append(parts,
				hint(keys.Select),
				hint(keys.Since),
				hint(keys.Until),
				hint(keys.Delete),
				hint(keys.Help),
				hint(keys.Quit),
			)
		}
	}
	left := strings.Join(parts, sep)
	right := styleDim.Render(Version)
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := max(m.width-leftWidth-rightWidth, 1)
	return left + strings.Repeat(" ", padding) + right
}

func hint(b key.Binding) string {
	return styleBold.Render(b.Help().Key) + " " + b.Help().Desc
}

func hintCustom(b key.Binding, desc string) string {
	return styleBold.Render(b.Help().Key) + " " + desc
}

func hintKeys(keyStr, desc string) string {
	return styleBold.Render(keyStr) + " " + desc
}

func onOff(on bool) string {
	if on {
		return styleActive.Render("on")
	}
	return styleDim.Render("off")
}
