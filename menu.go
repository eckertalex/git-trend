package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// menuEntry pairs a key binding with a verbose description for the keybindings
// overlay. The binding supplies the key string and the action when selected; the
// verbose desc is the full explanation shown only in the dialog (the footer uses
// the terser binding.Help().Desc instead).
type menuEntry struct {
	binding key.Binding
	desc    string
}

// menuEntries returns the ordered set of actions shown in the keybindings overlay.
func menuEntries() []menuEntry {
	return []menuEntry{
		{keys.Add, "Add author"},
		{keys.Me, "Add current git user"},
		{keys.Delete, "Delete selected"},
		{keys.Clear, "Clear all authors"},
		{keys.Total, "Toggle total line"},
		{keys.Top, "Fill top contributors"},
		{keys.Since, "Edit since date"},
		{keys.Until, "Edit until date"},
		{keys.Web, "Open in browser"},
		{keys.Export, "Export to git-trend.html"},
		{keys.Quit, "Quit"},
	}
}

// bindingKeyMsg synthesises a tea.KeyMsg for the first key in a binding, so
// selecting a menu item re-enters handleKey as if the user pressed that key.
func bindingKeyMsg(b key.Binding) tea.KeyMsg {
	ks := b.Keys()
	if len(ks) == 0 {
		return tea.KeyMsg{}
	}
	k := ks[0]
	switch k {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	}
}

func (m tuiModel) handleMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	entries := menuEntries()
	switch {
	case key.Matches(msg, keys.CtrlC):
		return m, tea.Quit
	case key.Matches(msg, keys.Esc), key.Matches(msg, keys.Help):
		m.mode = modeNormal
	case key.Matches(msg, keys.Up):
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case key.Matches(msg, keys.Down):
		if m.menuCursor < len(entries)-1 {
			m.menuCursor++
		}
	case key.Matches(msg, keys.Select):
		if m.menuCursor < len(entries) {
			m.mode = modeNormal
			return m.handleKey(bindingKeyMsg(entries[m.menuCursor].binding))
		}
	}
	return m, nil
}

func (m tuiModel) renderMenu() string {
	entries := menuEntries()
	if len(entries) == 0 {
		return ""
	}

	// Key column: wide enough for the longest key token.
	keyW := 0
	for _, e := range entries {
		if w := len(e.binding.Help().Key); w > keyW {
			keyW = w
		}
	}

	// Content width: key col + separator + longest description.
	contentW := 0
	for _, e := range entries {
		if w := keyW + 2 + len([]rune(e.desc)); w > contentW {
			contentW = w
		}
	}

	var sb strings.Builder
	for i, e := range entries {
		keyStr := e.binding.Help().Key
		desc := e.desc
		padding := strings.Repeat(" ", contentW-keyW-2-len([]rune(desc)))

		var line string
		if i == m.menuCursor {
			padded := lipgloss.NewStyle().
				Inline(true).Width(keyW).Render(keyStr)
			line = lipgloss.NewStyle().
				Foreground(selectedFg).Bold(true).Background(selectedBg).
				Render(padded + "  " + desc + padding)
		} else {
			padded := lipgloss.NewStyle().Foreground(colorFocus).Bold(true).
				Inline(true).Width(keyW).Render(keyStr)
			line = padded + "  " + desc
		}
		sb.WriteString(line)
		if i < len(entries)-1 {
			sb.WriteString("\n")
		}
	}

	rendered := borderFocused.Padding(1, 2).Render(sb.String())
	lines := strings.Split(rendered, "\n")
	if len(lines) > 0 {
		innerW := lipgloss.Width(lines[0]) - 2
		lines[0] = titleBorderLine("Keybindings", innerW, true, colorFocus)
	}
	return strings.Join(lines, "\n")
}
