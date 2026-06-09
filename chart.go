package main

import (
	"fmt"
	"time"

	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"
)

// dateLabelFormatter replaces DateTimeLabelFormatter so that ticks on the same
// calendar day always return an identical string. The built-in formatter returns
// "'26 06/01" for the first tick of a year and "06/01" for the next tick on the
// same day -- two different strings -- which defeats the duplicate-suppression in
// drawXLabel and prints the date twice.
func dateLabelFormatter() func(int, float64) string {
	var lastYear, lastDay, lastLabel string
	return func(i int, v float64) string {
		if i == 0 {
			lastYear, lastDay, lastLabel = "", "", ""
		}
		t := time.Unix(int64(v), 0).UTC()
		day := t.Format("01/02")
		year := t.Format("'06")
		if day == lastDay {
			return lastLabel // identical to last -- drawXLabel suppresses it
		}
		lastDay = day
		if year != lastYear {
			lastYear = year
			lastLabel = year + " " + day
		} else {
			lastLabel = day
		}
		return lastLabel
	}
}

func newChart(s []Series, samples []time.Time, maxY float64, w, h int, colors []lipgloss.AdaptiveColor) timeserieslinechart.Model {
	intFmt := func(_ int, v float64) string { return fmt.Sprintf("%d", int(v)) }
	lc := timeserieslinechart.New(w, h,
		timeserieslinechart.WithTimeRange(samples[0], samples[len(samples)-1]),
		timeserieslinechart.WithYRange(0, maxY),
		timeserieslinechart.WithXLabelFormatter(dateLabelFormatter()),
		timeserieslinechart.WithYLabelFormatter(intFmt),
	)

	for i, ser := range s {
		name := fmt.Sprintf("s%d", i)
		lc.SetDataSetStyle(name, lipgloss.NewStyle().Foreground(colors[i]))
		for j, v := range ser.Y {
			lc.PushDataSet(name, timeserieslinechart.TimePoint{Time: samples[j], Value: v})
		}
	}

	return lc
}

func yAndHeight(s []Series, height int) (maxY float64, chartH int) {
	for _, ser := range s {
		for _, v := range ser.Y {
			if v > maxY {
				maxY = v
			}
		}
	}
	if maxY == 0 {
		maxY = 1
	}
	chartH = max(min(int(maxY), height), 4)
	return maxY, chartH
}
