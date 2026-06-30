package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ── interaction ─────────────────────────────────────────────────────────────

// selectedAuthorLabel returns the identity label of the author highlighted in the
// authors list. ok is false when the selection is empty or the total line (which
// has no profile).
func (m tuiModel) selectedAuthorLabel() (label string, ok bool) {
	if len(m.series) == 0 {
		return "", false
	}
	sorted := sortedByCommits(m.series)
	i := m.authorsList.Index()
	if i < 0 || i >= len(sorted) {
		return "", false
	}
	if l := sorted[i].ser.Label; l != "total" {
		return l, true
	}
	return "", false
}

func (m *tuiModel) openAuthorDetail() {
	label, ok := m.selectedAuthorLabel()
	if !ok {
		return
	}
	m.detailAuthor = label
	m.rightView = viewAuthor
	m.sizeAuthorViewport()
	m.rebuildAuthorDetail()
	m.authorVP.GotoTop()
}

// syncAuthorDetail re-points the profile at the selected author while browsing the list.
func (m *tuiModel) syncAuthorDetail() {
	label, ok := m.selectedAuthorLabel()
	if !ok || label == m.detailAuthor {
		return
	}
	m.detailAuthor = label
	m.rebuildAuthorDetail()
	m.authorVP.GotoTop()
}

func (m *tuiModel) exitAuthorDetail() {
	m.rightView = viewChart
	m.detailAuthor = ""
	m.detailView = ""
}

func (m *tuiModel) sizeAuthorViewport() {
	m.authorVP.Width = m.chartWidth()
	m.authorVP.Height = max(m.panelHeight()-2, 1)
}

// rebuildAuthorDetail re-renders the profile into the viewport; no-op outside the author view.
func (m *tuiModel) rebuildAuthorDetail() {
	if m.rightView != viewAuthor {
		return
	}
	m.detailView = m.renderAuthorBody()
	m.authorVP.SetContent(m.detailView)
}

// ── stats ───────────────────────────────────────────────────────────────────

// streak is a run of active days with the dates it spanned.
type streak struct {
	days       int
	start, end time.Time
}

// streakStat is a record streak length plus every run that ties for it.
type streakStat struct {
	days  int
	spans []streak // chronological; len > 1 means the record was matched more than once
}

// weekdayStat aggregates one weekday across the author's history.
type weekdayStat struct {
	total      int // commits on that weekday
	activeDays int // distinct dates of that weekday with a commit
	peak       int // most commits on a single such date
}

// authorStats holds everything the profile page reports for one author, computed
// from their full commit history (independent of the active since/until range).
type authorStats struct {
	first, last   time.Time
	total         int
	pct           float64 // share of all repo commits
	activeDays    int
	avgPerWeek    float64
	peakDay       time.Time
	peakCount     int
	dailyStreak   streakStat        // record run of consecutive calendar days
	weekdayStreak streakStat        // record run where weekends don't break it
	weekdays      [7]weekdayStat    // indexed by time.Weekday (Sunday..Saturday)
	daily         map[time.Time]int // commits per calendar day (UTC-anchored)
}

func authorCommits(commits []Commit, label string) []Commit {
	var out []Commit
	for _, c := range commits {
		if ident(c) == label {
			out = append(out, c)
		}
	}
	return out
}

// groupCommitsByIdent indexes commits by author identity for O(1) profile lookups.
func groupCommitsByIdent(commits []Commit) map[string][]Commit {
	byIdent := make(map[string][]Commit)
	for _, c := range commits {
		id := ident(c)
		byIdent[id] = append(byIdent[id], c)
	}
	return byIdent
}

// dayOf anchors a commit time to midnight UTC on its own calendar date, giving a
// comparable key that is free of DST so day-to-day arithmetic is always exact.
func dayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// daysBetween counts whole days from a to b; both must be dayOf values.
func daysBetween(a, b time.Time) int {
	return int(b.Sub(a) / (24 * time.Hour))
}

func computeAuthorStats(commits []Commit, repoTotal int) authorStats {
	var s authorStats
	if len(commits) == 0 {
		return s
	}

	s.total = len(commits)
	if repoTotal > 0 {
		s.pct = float64(s.total) / float64(repoTotal) * 100
	}

	s.daily = make(map[time.Time]int)
	for _, c := range commits {
		s.daily[dayOf(c.When)]++
	}
	for d, n := range s.daily {
		ws := &s.weekdays[d.Weekday()]
		ws.total += n
		ws.activeDays++
		if n > ws.peak {
			ws.peak = n
		}
	}

	days := sortedDays(s.daily)
	s.activeDays = len(days)
	s.first = days[0]
	s.last = days[len(days)-1]

	s.dailyStreak = topStreaks(calendarRuns(days))
	s.weekdayStreak = topStreaks(weekdayRuns(days))

	spanDays := daysBetween(s.first, s.last) + 1
	weeks := float64(spanDays) / 7
	if weeks < 1 {
		weeks = 1
	}
	s.avgPerWeek = float64(s.total) / weeks

	s.peakDay, s.peakCount = peakDay(s.daily)

	return s
}

func sortedDays(daily map[time.Time]int) []time.Time {
	days := make([]time.Time, 0, len(daily))
	for d := range daily {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	return days
}

// calendarRuns returns every maximal run of consecutive calendar days.
func calendarRuns(days []time.Time) []streak {
	return runsBy(days, func(a, b time.Time) bool { return daysBetween(a, b) == 1 })
}

// weekdayRuns returns every maximal run where the only gaps are weekends. A missed
// weekday breaks the run; weekend commits still count toward it.
func weekdayRuns(days []time.Time) []streak {
	return runsBy(days, onlyWeekendBetween)
}

// runsBy groups sorted days into maximal runs where each consecutive pair
// satisfies connected(prev, cur).
func runsBy(days []time.Time, connected func(a, b time.Time) bool) []streak {
	var runs []streak
	var cur streak
	for i, d := range days {
		if i > 0 && connected(days[i-1], d) {
			cur.days++
			cur.end = d
		} else {
			if cur.days > 0 {
				runs = append(runs, cur)
			}
			cur = streak{days: 1, start: d, end: d}
		}
	}
	if cur.days > 0 {
		runs = append(runs, cur)
	}
	return runs
}

// topStreaks returns the record length and every run that ties for it.
func topStreaks(runs []streak) streakStat {
	var ss streakStat
	for _, r := range runs {
		if r.days > ss.days {
			ss.days = r.days
		}
	}
	for _, r := range runs {
		if r.days == ss.days {
			ss.spans = append(ss.spans, r)
		}
	}
	return ss
}

// onlyWeekendBetween reports whether every calendar day strictly between a and b
// is a weekend day. Consecutive days (nothing between) trivially qualify.
func onlyWeekendBetween(a, b time.Time) bool {
	for d := a.AddDate(0, 0, 1); d.Before(b); d = d.AddDate(0, 0, 1) {
		if wd := d.Weekday(); wd != time.Saturday && wd != time.Sunday {
			return false
		}
	}
	return true
}

func peakDay(daily map[time.Time]int) (day time.Time, count int) {
	for d, n := range daily {
		// On ties prefer the earlier day for determinism.
		if n > count || (n == count && !day.IsZero() && d.Before(day)) {
			day, count = d, n
		}
	}
	return day, count
}

// ── rendering ─────────────────────────────────────────────────────────────--

// authorChartHeight is the fixed height of the per-author line chart. The body
// flows at its natural height; the profile viewport scrolls when it overflows.
const authorChartHeight = 8

// authorProfileSource returns the commit set the profile is built from: the
// unfiltered history so the profile is author-centric and ignores the date range.
// Falls back to the loaded (filtered) set until the full history arrives.
func (m tuiModel) authorProfileSource() []Commit {
	if m.fullCommits != nil {
		return m.fullCommits
	}
	return m.allCommits
}

// profileCommits returns the drilled-in author's commits via the prebuilt index,
// falling back to a scan of source until the full history is loaded.
func (m tuiModel) profileCommits(source []Commit) []Commit {
	if m.commitsByAuthor != nil {
		return m.commitsByAuthor[m.detailAuthor]
	}
	return authorCommits(source, m.detailAuthor)
}

// renderAuthorBody renders the profile body, or "" when the author has no commits
// (the panel then shows its empty state).
func (m tuiModel) renderAuthorBody() string {
	source := m.authorProfileSource()
	commits := m.profileCommits(source)
	if len(commits) == 0 {
		return ""
	}
	innerW := m.chartWidth()
	stats := computeAuthorStats(commits, len(source))

	var lines []string
	_, email := parseNameEmail(m.detailAuthor)
	lines = append(lines, styleDim.Render(email), "")
	lines = append(lines, statsLines(stats, innerW)...)
	lines = append(lines, "")

	lines = append(lines, weekdayLines(stats)...)
	lines = append(lines, "")

	lines = append(lines, styleBold.Render("Commits over time"))
	lines = append(lines, m.renderAuthorChart(commits))
	lines = append(lines, "")

	lines = append(lines, styleBold.Render("Activity"))
	lines = append(lines, renderContributionGraph(stats, innerW))

	return strings.Join(lines, "\n")
}

func statsLines(s authorStats, innerW int) []string {
	col := max(innerW/2, 24)
	kv := func(label, val string) string {
		return styleDim.Render(label+":") + " " + val
	}
	row := func(a, b string) string {
		if b == "" {
			return a
		}
		return lipgloss.NewStyle().Width(col).Inline(true).Render(a) + b
	}
	// Padded label for the streak block so the values line up.
	skv := func(label, val string) string {
		return styleDim.Render(fmt.Sprintf("%-15s", label+":")) + val
	}
	lines := []string{
		row(kv("First", date(s.first)), kv("Last", date(s.last))),
		row(kv("Commits", fmt.Sprintf("%d (%.0f%% of repo)", s.total, s.pct)),
			kv("Active days", fmt.Sprintf("%d", s.activeDays))),
		row(kv("Avg/wk", fmt.Sprintf("%.1f", s.avgPerWeek)),
			kv("Peak", fmt.Sprintf("%s (%d)", date(s.peakDay), s.peakCount))),
		"",
		skv("Daily streak", streakStatSpan(s.dailyStreak)),
		skv("Weekday streak", streakStatSpan(s.weekdayStreak)),
	}
	return lines
}

// streakStatSpan describes the record length and the date ranges that achieved it.
// When more than a couple of ranges tie, it lists the first few and counts the rest.
func streakStatSpan(ss streakStat) string {
	if ss.days <= 1 {
		// 0 = none; 1 = the author never had two consecutive active days, so the
		// individual dates aren't worth enumerating.
		return fmt.Sprintf("%dd", ss.days)
	}
	const showMax = 2
	var parts []string
	for i, sp := range ss.spans {
		if i >= showMax {
			break
		}
		parts = append(parts, fmt.Sprintf("%s → %s", date(sp.start), date(sp.end)))
	}
	out := fmt.Sprintf("%dd  %s", ss.days, strings.Join(parts, ", "))
	if extra := len(ss.spans) - showMax; extra > 0 {
		out += fmt.Sprintf(" … (+%d more)", extra)
	}
	return out
}

// weekdayLines renders a per-weekday breakdown with a bar, total, average commits
// per active day, and single-day peak. Returns nil when there is no activity.
func weekdayLines(s authorStats) []string {
	maxTotal := 0
	for _, ws := range s.weekdays {
		if ws.total > maxTotal {
			maxTotal = ws.total
		}
	}
	if maxTotal == 0 {
		return nil
	}

	const barMax = 12
	order := []time.Weekday{
		time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
		time.Friday, time.Saturday, time.Sunday,
	}
	lines := []string{styleBold.Render("By weekday")}
	for _, wd := range order {
		ws := s.weekdays[wd]
		barLen := ws.total * barMax / maxTotal
		bar := lipgloss.NewStyle().Foreground(heatScale[3]).Render(strings.Repeat("█", barLen)) +
			strings.Repeat(" ", barMax-barLen)
		avg := 0.0
		if ws.activeDays > 0 {
			avg = float64(ws.total) / float64(ws.activeDays)
		}
		stat := styleDim.Render(fmt.Sprintf("%4d   avg %.1f   peak %d", ws.total, avg, ws.peak))
		lines = append(lines, fmt.Sprintf("%s %s %s", wd.String()[:3], bar, stat))
	}
	return lines
}

func date(t time.Time) string {
	if t.IsZero() {
		return "–"
	}
	return t.Format("2006-01-02")
}

// renderAuthorChart draws the author's cumulative commit line over their own full
// history, on a time grid spanning their first to last commit (not the chart's
// date-filtered grid), reusing the shared chart helpers.
func (m tuiModel) renderAuthorChart(commits []Commit) string {
	w := m.chartWidth()
	sorted := sortByTime(commits)
	start, end := sorted[0].When, sorted[len(sorted)-1].When
	if !end.After(start) {
		end = start.Add(time.Hour) // single-day author: keep a valid x range
	}
	samples := sampleTimes(start, end, max(w, 2))

	name, _ := parseNameEmail(m.detailAuthor)
	ser := []Series{{Label: name, Y: cumulative(sorted, samples)}}

	color := palette[m.colorByLabel[m.detailAuthor]%len(palette)]
	maxY, chartH := yAndHeight(ser, authorChartHeight)
	lc := newChart(ser, samples, maxY, w, chartH, []lipgloss.AdaptiveColor{color})
	lc.DrawBrailleAll()
	return lc.View()
}

// renderContributionGraph draws a GitHub-style heatmap: 7 weekday rows by week
// columns, colored by daily commit volume. When the active span is wider than the
// panel, it shows the most recent weeks that fit and captions the visible window.
func renderContributionGraph(s authorStats, innerW int) string {
	const labelW = 4 // "Mon " column

	weekStart := func(d time.Time) time.Time { return d.AddDate(0, 0, -int(d.Weekday())) }
	lastWeek := weekStart(s.last)
	totalWeeks := daysBetween(weekStart(s.first), lastWeek)/7 + 1

	cols := max(innerW-labelW, 1)
	truncated := totalWeeks > cols
	if !truncated {
		cols = totalWeeks
	}
	firstWeek := lastWeek.AddDate(0, 0, -7*(cols-1))

	rowLabels := map[int]string{1: "Mon", 3: "Wed", 5: "Fri"}
	var rows []string
	for wd := range 7 {
		var b strings.Builder
		fmt.Fprintf(&b, "%-3s ", rowLabels[wd])
		for c := 0; c < cols; c++ {
			day := firstWeek.AddDate(0, 0, 7*c+wd)
			color := heatScale[heatBucket(s.daily[day])]
			b.WriteString(lipgloss.NewStyle().Foreground(color).Render("■"))
		}
		rows = append(rows, b.String())
	}

	caption := fmt.Sprintf("%s – %s", date(firstWeek), date(s.last))
	if truncated {
		caption = fmt.Sprintf("last %d weeks (%s)", cols, caption)
	}
	rows = append(rows, styleDim.Render(caption))
	return strings.Join(rows, "\n")
}

func heatBucket(n int) int {
	switch {
	case n <= 0:
		return 0
	case n == 1:
		return 1
	case n <= 3:
		return 2
	case n <= 5:
		return 3
	default:
		return 4
	}
}
