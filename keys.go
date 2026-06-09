package main

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ── keymap ────────────────────────────────────────────────────────────────────

type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Select  key.Binding // enter or space -- normal-mode confirm/activate
	Confirm key.Binding // enter only -- input confirmation
	Esc     key.Binding
	Add     key.Binding
	Delete  key.Binding
	Clear   key.Binding
	Total   key.Binding
	Top     key.Binding
	Since   key.Binding
	Until   key.Binding
	Me      key.Binding
	Web     key.Binding
	Export  key.Binding
	Help    key.Binding
	Quit    key.Binding
	CtrlC   key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:    key.NewBinding(key.WithKeys("left", "h", "shift+tab"), key.WithHelp("←/h", "prev box")),
	Right:   key.NewBinding(key.WithKeys("right", "l", "tab"), key.WithHelp("→/l", "next box")),
	Select:  key.NewBinding(key.WithKeys("enter", " "), key.WithHelp("<enter>", "edit")),
	Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	Esc:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("<esc>", "cancel")),
	Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
	Delete:  key.NewBinding(key.WithKeys("d", "backspace"), key.WithHelp("d", "del")),
	Clear:   key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "clear")),
	Total:   key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "total")),
	Top:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "top")),
	Since:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "since")),
	Until:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "until")),
	Me:      key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "me")),
	Web:     key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "web")),
	Export:  key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	CtrlC:   key.NewBinding(key.WithKeys("ctrl+c")),
}

func (m tuiModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = ""
	switch {
	case key.Matches(msg, keys.CtrlC), key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Left):
		m.focusedBox = (m.focusedBox + numBoxes - 1) % numBoxes

	case key.Matches(msg, keys.Right):
		m.focusedBox = (m.focusedBox + 1) % numBoxes

	case key.Matches(msg, keys.Help):
		m.mode = modeMenu
		m.menuCursor = 0
		return m, nil

	case key.Matches(msg, keys.Add):
		if len(m.authors) >= m.effectiveCapacity() {
			m.statusMsg = fmt.Sprintf("warning: at capacity (%d/%d)", m.usedSeriesCount(), maxSeries)
			return m, nil
		}
		m.mode = modeAddAuthor
		m.authorInput.SetValue("")
		cmd := m.authorInput.Focus()
		return m, cmd

	case key.Matches(msg, keys.Me):
		return m, fetchCurrentUser()

	case key.Matches(msg, keys.Total):
		if !m.showTotal && len(m.authors) >= maxSeries {
			m.statusMsg = fmt.Sprintf("warning: at capacity (%d/%d)", m.usedSeriesCount(), maxSeries)
			return m, nil
		}
		m.showTotal = !m.showTotal
		m.recompute()

	case key.Matches(msg, keys.Top):
		m.fillTopContributors()
		m.recompute()

	case key.Matches(msg, keys.Clear):
		m.clearAuthors()
		m.recompute()

	case key.Matches(msg, keys.Web):
		series, samples, start, end := m.series, m.samples, m.start, m.end
		return m, func() tea.Msg {
			return webMsg{openWebFromTUI(series, samples, start, end)}
		}

	case key.Matches(msg, keys.Export):
		series, samples, start, end := m.series, m.samples, m.start, m.end
		return m, func() tea.Msg {
			path, err := exportWebFromTUI(series, samples, start, end)
			return exportedMsg{path, err}
		}

	case key.Matches(msg, keys.Up):
		var cmd tea.Cmd
		switch m.focusedBox {
		case boxAuthors:
			m.authorsList, cmd = m.authorsList.Update(msg)
		case boxTimeFrames:
			m.timeRangeList, cmd = m.timeRangeList.Update(msg)
		}
		return m, cmd

	case key.Matches(msg, keys.Down):
		var cmd tea.Cmd
		switch m.focusedBox {
		case boxAuthors:
			m.authorsList, cmd = m.authorsList.Update(msg)
		case boxTimeFrames:
			m.timeRangeList, cmd = m.timeRangeList.Update(msg)
		}
		return m, cmd

	case key.Matches(msg, keys.Delete):
		switch m.focusedBox {
		case boxAuthors:
			if len(m.series) > 0 {
				c := m.authorsList.Index()
				sorted := sortedByCommits(m.series)
				if c < len(sorted) {
					m.removeSeries(sorted[c].ser)
				}
			}
		case boxTimeFrames:
			if m.timeRangeList.Index() == 0 {
				m.since = ""
			} else {
				m.until = ""
			}
			m.syncTimeRange()
			m.loading = true
			return m, tea.Batch(fetchCommits(m.since, m.until), m.spinner.Tick)
		}

	case key.Matches(msg, keys.Select):
		if m.focusedBox == boxTimeFrames {
			field := modeEditSince
			if m.timeRangeList.Index() != 0 {
				field = modeEditUntil
			}
			return m, m.beginDateEdit(field)
		}

	case key.Matches(msg, keys.Since):
		return m, m.beginDateEdit(modeEditSince)

	case key.Matches(msg, keys.Until):
		return m, m.beginDateEdit(modeEditUntil)
	}
	return m, nil
}

func (m tuiModel) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.CtrlC):
		return m, tea.Quit

	case key.Matches(msg, keys.Confirm):
		pattern := strings.TrimSpace(m.authorInput.Value())
		if pattern != "" {
			if _, err := regexp.Compile(pattern); err != nil {
				m.statusMsg = fmt.Sprintf("error: invalid author pattern %q", pattern)
				m.mode = modeNormal
				m.authorInput.Blur()
				return m, nil
			}
			expanded := expandAuthors([]string{pattern}, m.allCommits)
			if len(expanded) > 1 {
				sortByCommitCount(expanded, m.allCommits)
			}
			added, overflow := 0, 0
			for _, p := range expanded {
				if slices.Contains(m.authors, p) {
					continue
				}
				if len(m.authors) >= m.effectiveCapacity() {
					overflow++
					continue
				}
				m.authors = append(m.authors, p)
				added++
			}
			if overflow > 0 {
				m.statusMsg = fmt.Sprintf("warning: added %d of %d matches (at capacity)", added, added+overflow)
			}
			m.recompute()
		}
		m.mode = modeNormal
		m.authorInput.Blur()
		return m, nil

	case key.Matches(msg, keys.Esc):
		m.mode = modeNormal
		m.authorInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.authorInput, cmd = m.authorInput.Update(msg)
	return m, cmd
}

func (m tuiModel) handleDateInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.CtrlC):
		return m, tea.Quit

	case key.Matches(msg, keys.Confirm):
		val := strings.TrimSpace(m.dateInput.Value())
		if m.mode == modeEditSince {
			m.since = val
		} else {
			m.until = val
		}
		m.syncTimeRange()
		m.mode = modeNormal
		m.dateInput.Blur()
		m.loading = true
		return m, tea.Batch(fetchCommits(m.since, m.until), m.spinner.Tick)

	case key.Matches(msg, keys.Esc):
		m.mode = modeNormal
		m.dateInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.dateInput, cmd = m.dateInput.Update(msg)
	return m, cmd
}
