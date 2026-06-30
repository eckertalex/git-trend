package main

import (
	"fmt"
	"regexp"
	"sort"
	"time"
)

type Series struct {
	Label string
	Y     []float64
}

type BuildOptions struct {
	Authors   []string
	ShowTop   bool
	ShowTotal bool
	Width     int
}

// Overflow reports lines that didn't fit the maxSeries cap, so callers can warn.
type Overflow struct {
	Authors int  // explicit authors omitted for capacity
	Total   bool // --total requested but no slot left
}

// Chart is the result of a Build call.
type Chart struct {
	Series  []Series
	Samples []time.Time // x grid the Y values were computed against
	Start   time.Time   // true first-commit time (not the -1ns grid start)
	End     time.Time   // true last-commit time
	Dropped Overflow
}

// Build assembles the series in priority order: explicit authors (kept by commit
// count on overflow) win the slots first, then --total, then --top fill.
func Build(commits []Commit, opts BuildOptions) (Chart, error) {
	if len(commits) == 0 {
		return Chart{}, nil
	}

	width := max(opts.Width, 2)

	sorted := sortByTime(commits)

	start := sorted[0].When
	end := sorted[len(sorted)-1].When

	samples := sampleTimes(start, end, width)

	capacity := maxSeries
	seen := make(map[string]bool)
	var dropped Overflow

	var authorSeries []Series
	if len(opts.Authors) > 0 {
		groups, err := matchAuthors(sorted, opts.Authors)
		if err != nil {
			return Chart{}, err
		}

		// dedup by identity, preserving first-match order.
		// Multiple patterns may match the same identity; keep only the first group.
		authorSeen := make(map[string]bool)
		var uniq []authorGroup
		for _, g := range groups {
			if authorSeen[g.label] {
				continue
			}

			authorSeen[g.label] = true
			uniq = append(uniq, g)
		}

		kept := uniq
		if len(uniq) > capacity {
			kept = topByCommits(uniq, capacity)
			dropped.Authors = len(uniq) - capacity
		}

		for _, g := range kept {
			seen[g.label] = true
			authorSeries = append(authorSeries, Series{Label: g.label, Y: cumulative(g.commits, samples)})
		}

		capacity -= len(kept)
	}

	var series []Series
	if opts.ShowTotal {
		if capacity > 0 {
			series = append(series, Series{Label: "total", Y: cumulative(sorted, samples)})
			capacity--
		} else {
			dropped.Total = true
		}
	}
	series = append(series, authorSeries...)

	if opts.ShowTop && capacity > 0 {
		ranked := rankByCommitCount(sorted)

		added := 0
		for _, g := range ranked {
			if added >= capacity {
				break
			}
			if seen[g.label] {
				continue
			}
			seen[g.label] = true
			series = append(series, Series{Label: g.label, Y: cumulative(g.commits, samples)})
			added++
		}
	}

	return Chart{
		Series:  series,
		Samples: samples,
		Start:   start,
		End:     end,
		Dropped: dropped,
	}, nil
}

// topByCommits keeps the n highest-commit groups, preserving their original order.
func topByCommits(groups []authorGroup, n int) []authorGroup {
	if n >= len(groups) {
		return groups
	}
	idx := make([]int, len(groups))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		return len(groups[idx[a]].commits) > len(groups[idx[b]].commits)
	})
	keep := make(map[int]bool, n)
	for _, i := range idx[:n] {
		keep[i] = true
	}
	out := make([]authorGroup, 0, n)
	for i, g := range groups {
		if keep[i] {
			out = append(out, g)
		}
	}
	return out
}

type authorGroup struct {
	label   string
	commits []Commit
}

// matches against "Name <email>", mirroring git --author
// One term can match multiple authors.
func matchAuthors(sorted []Commit, terms []string) (groups []authorGroup, err error) {
	for _, term := range terms {
		re, err := regexp.Compile(term)
		if err != nil {
			return nil, fmt.Errorf("invalid --author pattern %q: %w", term, err)
		}
		groups = append(groups, groupByAuthor(filter(sorted, re))...)
	}
	return groups, nil
}

func groupByAuthor(commits []Commit) []authorGroup {
	byIdent := make(map[string][]Commit)
	var order []string
	for _, c := range commits {
		id := ident(c)
		if _, seen := byIdent[id]; !seen {
			order = append(order, id)
		}
		byIdent[id] = append(byIdent[id], c)
	}

	groups := make([]authorGroup, 0, len(order))
	for _, id := range order {
		groups = append(groups, authorGroup{label: id, commits: byIdent[id]})
	}
	return groups
}

// sortByTime returns a copy of commits ordered oldest-first, leaving the input
// untouched.
func sortByTime(commits []Commit) []Commit {
	sorted := make([]Commit, len(commits))
	copy(sorted, commits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].When.Before(sorted[j].When)
	})
	return sorted
}

func sampleTimes(start, end time.Time, n int) []time.Time {
	times := make([]time.Time, n)
	span := end.Sub(start)
	for i := range n {
		frac := float64(i) / float64(n-1)
		times[i] = start.Add(time.Duration(frac * float64(span)))
	}
	return times
}

func cumulative(commits []Commit, samples []time.Time) []float64 {
	y := make([]float64, len(samples))
	idx := 0
	count := 0
	for i, t := range samples {
		// Use strict < for the first sample so y[0] is always 0,
		// even when a commit lands exactly on samples[0].
		limit := t
		if i == 0 {
			limit = t.Add(-1)
		}
		for idx < len(commits) && !commits[idx].When.After(limit) {
			count++
			idx++
		}
		y[i] = float64(count)
	}
	return y
}

func filter(commits []Commit, re *regexp.Regexp) []Commit {
	var matched []Commit
	for _, c := range commits {
		if re.MatchString(ident(c)) {
			matched = append(matched, c)
		}
	}
	return matched
}

func ident(c Commit) string {
	return fmt.Sprintf("%s <%s>", c.AuthorName, c.AuthorEmail)
}

// rankByCommitCount groups commits by author identity and returns the groups
// sorted descending by commit count (stable, preserving insertion order on ties).
func rankByCommitCount(commits []Commit) []authorGroup {
	ranked := groupByAuthor(commits)
	sort.SliceStable(ranked, func(i, j int) bool {
		return len(ranked[i].commits) > len(ranked[j].commits)
	})
	return ranked
}
