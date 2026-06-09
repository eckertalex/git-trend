package main

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.authorsList.SetSize(leftPanelWidth-2, m.authorsPanelHeight())
		m.recompute()
		return m, nil

	case commitsMsg:
		m.loading = false
		if msg.err != nil {
			m.buildErr = msg.err
			return m, nil
		}
		m.allCommits = msg.commits
		if !m.initialized {
			m.initialized = true
			var dropped int
			m.authors, dropped = fitAuthors(m.authors, msg.commits, m.effectiveCapacity())
			if dropped > 0 {
				m.statusMsg = fmt.Sprintf("warning: %d more authors matched, trimmed to %d", dropped, len(m.authors))
			}
			if m.pendingTopFill {
				m.pendingTopFill = false
				m.fillTopContributors()
			}
		}
		m.recompute()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case currentUserMsg:
		if msg.err == nil && msg.identity != "" {
			pattern := identityPattern(msg.identity)
			if slices.Contains(m.authors, pattern) {
				return m, nil
			}
			if len(m.authors) < m.effectiveCapacity() {
				m.authors = append(m.authors, pattern)
				m.recompute()
			} else {
				m.statusMsg = fmt.Sprintf("warning: at capacity (%d/%d)", m.usedSeriesCount(), maxSeries)
			}
		}
		return m, nil

	case webMsg:
		if msg.err != nil {
			m.statusMsg = "error: " + msg.err.Error()
		}
		return m, nil

	case exportedMsg:
		if msg.err != nil {
			m.statusMsg = "error: " + msg.err.Error()
		} else {
			m.statusMsg = "exported to " + msg.path
		}
		return m, nil

	case tea.KeyMsg:
		// ctrl+c is handled universally before mode routing
		if key.Matches(msg, keys.CtrlC) {
			return m, tea.Quit
		}
		switch m.mode {
		case modeAddAuthor:
			return m.handleInputKey(msg)
		case modeEditSince, modeEditUntil:
			return m.handleDateInputKey(msg)
		case modeMenu:
			return m.handleMenuKey(msg)
		default:
			return m.handleKey(msg)
		}
	}
	return m, nil
}

// recompute validates authors, rebuilds chart data from allCommits, updates the
// author list, and caches a rendered chart string. Call whenever data or authors
// change -- not on every frame.
func (m *tuiModel) recompute() {
	if m.width == 0 || m.height == 0 {
		return
	}
	chartW := m.chartWidth()
	if chartW < 10 {
		return
	}

	// Drop patterns that don't compile.
	var valid []string
	for _, p := range m.authors {
		if _, err := regexp.Compile(p); err != nil {
			m.statusMsg = fmt.Sprintf("error: invalid author pattern %q", p)
		} else {
			valid = append(valid, p)
		}
	}
	m.authors = valid

	chart, err := Build(m.allCommits, BuildOptions{
		Authors:   m.authors,
		ShowTotal: m.showTotal,
		Width:     chartW,
	})
	m.buildErr = err
	m.series = chart.Series
	m.samples = chart.Samples
	m.start = chart.Start
	m.end = chart.End

	// Assign stable palette colors to any new series labels.
	for _, ser := range m.series {
		if _, ok := m.colorByLabel[ser.Label]; !ok {
			m.colorByLabel[ser.Label] = m.nextColor % len(palette)
			m.nextColor++
		}
	}

	// Sync author list items.
	sorted := sortedByCommits(m.series)
	items := make([]list.Item, len(sorted))
	for i, s := range sorted {
		name, _ := parseNameEmail(s.ser.Label)
		items[i] = authorItem{
			name:     name,
			commits:  int(lastY(s.ser)),
			colorIdx: m.colorByLabel[s.ser.Label],
		}
	}
	m.authorsList.SetItems(items)

	m.syncTimeRange()
	m.chartView = m.renderChart()
}

// renderChart builds and renders the chart for the current series, returning it
// as a string. The result is cached in m.chartView by recompute.
func (m tuiModel) renderChart() string {
	if err := m.buildErr; err != nil || len(m.series) == 0 || len(m.samples) == 0 {
		return ""
	}
	chartW := m.chartWidth()
	if chartW < 10 {
		return ""
	}
	chartH := max(m.panelHeight()-2, 4)
	maxY, h := yAndHeight(m.series, chartH)

	colors := make([]lipgloss.AdaptiveColor, len(m.series))
	for i, ser := range m.series {
		colors[i] = palette[m.colorByLabel[ser.Label]]
	}

	lc := newChart(m.series, m.samples, maxY, chartW, h, colors)
	lc.DrawBrailleAll()
	return lc.View()
}

// compactDelegate is the default list delegate with the inter-item blank line
// removed and the left indent trimmed for a denser panel.
func compactDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.SetSpacing(0)
	d.Styles.NormalTitle = d.Styles.NormalTitle.PaddingLeft(1)
	d.Styles.NormalDesc = d.Styles.NormalDesc.PaddingLeft(1)
	d.Styles.DimmedTitle = d.Styles.DimmedTitle.PaddingLeft(1)
	d.Styles.DimmedDesc = d.Styles.DimmedDesc.PaddingLeft(1)
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.PaddingLeft(0)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.PaddingLeft(0)
	return d
}

type authorItem struct {
	name     string
	commits  int
	colorIdx int
}

func (a authorItem) Title() string {
	dot := lipgloss.NewStyle().Foreground(palette[a.colorIdx%len(palette)]).Render("●")
	return dot + " " + a.name
}
func (a authorItem) Description() string { return fmt.Sprintf("%d commits", a.commits) }
func (a authorItem) FilterValue() string { return a.name }

type timeRangeItem struct {
	label       string
	value       string
	placeholder string
}

func (t timeRangeItem) Title() string { return t.label }
func (t timeRangeItem) Description() string {
	if t.value != "" {
		return t.value
	}
	return t.placeholder
}
func (t timeRangeItem) FilterValue() string { return t.label }

// syncTimeRange updates the time range list items to reflect the current
// m.since / m.until values while preserving the cursor position.
func (m *tuiModel) syncTimeRange() {
	idx := m.timeRangeList.Index()
	m.timeRangeList.SetItems([]list.Item{
		timeRangeItem{label: "since", value: m.since, placeholder: "all time"},
		timeRangeItem{label: "until", value: m.until, placeholder: "now"},
	})
	m.timeRangeList.Select(idx)
}

func (m *tuiModel) removeSeries(ser Series) {
	if ser.Label == "total" {
		m.showTotal = false
		m.recompute()
		return
	}
	m.authors = sliceRemove(m.authors, identityPattern(ser.Label))
	m.recompute()
}

func (m *tuiModel) clearAuthors() {
	m.authors = nil
	m.showTotal = false
}

func (m *tuiModel) fillTopContributors() {
	capacity := m.effectiveCapacity() - len(m.authors)
	if capacity <= 0 || len(m.allCommits) == 0 {
		return
	}

	seen := make(map[string]bool)
	for _, a := range m.authors {
		seen[a] = true
	}

	added := 0
	for _, g := range rankByCommitCount(m.allCommits) {
		if added >= capacity {
			break
		}
		pattern := identityPattern(g.label)
		if seen[pattern] {
			continue
		}
		seen[pattern] = true
		m.authors = append(m.authors, pattern)
		added++
	}
}

func (m *tuiModel) beginDateEdit(field mode) tea.Cmd {
	value := m.since
	placeholder := sinceHint
	if field == modeEditUntil {
		value = m.until
		placeholder = untilHint
	}
	m.mode = field
	m.dateInput.Placeholder = placeholder
	m.dateInput.SetValue(value)
	return m.dateInput.Focus()
}

func authorCapacity(showTotal bool) int {
	if showTotal {
		return maxSeries - 1
	}
	return maxSeries
}

func (m tuiModel) effectiveCapacity() int {
	return authorCapacity(m.showTotal)
}

func (m tuiModel) usedSeriesCount() int {
	n := len(m.authors)
	if m.showTotal {
		n++
	}
	return n
}

// identityPattern returns an anchored regex that matches exactly one identity.
func identityPattern(id string) string {
	return "^" + regexp.QuoteMeta(id) + "$"
}

// expandAuthors resolves patterns to per-person email patterns; keeps unmatched
// patterns as-is for future commits.
func expandAuthors(patterns []string, commits []Commit) []string {
	seen := make(map[string]bool)
	var result []string
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			if !seen[pattern] {
				seen[pattern] = true
				result = append(result, pattern)
			}
			continue
		}
		groups := groupByAuthor(filter(commits, re))
		if len(groups) == 0 {
			if !seen[pattern] {
				seen[pattern] = true
				result = append(result, pattern)
			}
			continue
		}
		for _, g := range groups {
			p := identityPattern(g.label)
			if !seen[p] {
				seen[p] = true
				result = append(result, p)
			}
		}
	}
	return result
}

func fitAuthors(patterns []string, commits []Commit, capacity int) (kept []string, dropped int) {
	expanded := expandAuthors(patterns, commits)
	sortByCommitCount(expanded, commits)
	if len(expanded) <= capacity {
		return expanded, 0
	}
	return expanded[:capacity], len(expanded) - capacity
}

func sortByCommitCount(authors []string, commits []Commit) {
	countByPattern := make(map[string]int)
	for _, c := range commits {
		countByPattern[identityPattern(ident(c))]++
	}
	sort.SliceStable(authors, func(i, j int) bool {
		return countByPattern[authors[i]] > countByPattern[authors[j]]
	})
}

type seriesWithIdx struct {
	ser Series
	idx int
}

func sortedByCommits(series []Series) []seriesWithIdx {
	out := make([]seriesWithIdx, len(series))
	for i, s := range series {
		out[i] = seriesWithIdx{s, i}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return lastY(out[i].ser) > lastY(out[j].ser)
	})
	return out
}

func lastY(s Series) float64 {
	if len(s.Y) == 0 {
		return 0
	}
	return s.Y[len(s.Y)-1]
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func sliceRemove(ss []string, s string) []string {
	out := ss[:0:0]
	for _, v := range ss {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

func parseNameEmail(label string) (name, email string) {
	i := strings.LastIndex(label, " <")
	if i < 0 {
		return label, ""
	}
	return label[:i], strings.TrimSuffix(label[i+2:], ">")
}
