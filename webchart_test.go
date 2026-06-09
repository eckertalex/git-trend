package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLegendLabel(t *testing.T) {
	tests := []struct {
		name string
		in   Series
		want string
	}{
		{
			name: "non-zero start shows start and end",
			in:   Series{Label: "alice", Y: []float64{3, 5, 9}},
			want: "alice (3 → 9)",
		},
		{
			name: "zero start shows only end",
			in:   Series{Label: "bob", Y: []float64{0, 0, 4}},
			want: "bob (4)",
		},
		{
			name: "empty series falls back to label",
			in:   Series{Label: "total", Y: nil},
			want: "total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := legendLabel(tt.in); got != tt.want {
				t.Errorf("legendLabel(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRenderHTMLContainsSeriesLabels(t *testing.T) {
	s := []Series{
		{Label: "Alice <alice@example.com>", Y: []float64{0, 1, 2, 3}},
		{Label: "Bob <bob@example.com>", Y: []float64{0, 0, 1, 2}},
	}
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	samples := sampleTimes(start.Add(-1), end, len(s[0].Y))

	var buf bytes.Buffer
	if err := RenderHTML(s, samples, start, end, "", &buf); err != nil {
		t.Fatalf("RenderHTML returned error: %v", err)
	}

	html := buf.String()
	for _, want := range []string{"Alice", "Bob", "git-trend", "echarts"} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
}

func TestRenderHTMLEmptySeries(t *testing.T) {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)

	var buf bytes.Buffer
	if err := RenderHTML(nil, nil, start, end, "", &buf); err != nil {
		t.Fatalf("RenderHTML returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty HTML output even with no series")
	}
}

func TestWriteChartExplicitPath(t *testing.T) {
	t.Chdir(initRepo(t))

	path := t.TempDir() + "/chart.html"
	got, err := writeChart(BuildOptions{ShowTotal: true}, "", "", path)
	if err != nil {
		t.Fatalf("writeChart returned error: %v", err)
	}
	if got != path {
		t.Errorf("writeChart returned path %q, want %q", got, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("HTML file not written: %v", err)
	}
	if len(content) == 0 {
		t.Error("HTML file is empty")
	}
}

func TestWriteChartTempFile(t *testing.T) {
	t.Chdir(initRepo(t))

	got, err := writeChart(BuildOptions{ShowTotal: true}, "", "", "")
	if err != nil {
		t.Fatalf("writeChart returned error: %v", err)
	}
	if got == "git-trend.html" {
		t.Error("writeChart with empty out should use a temp file, not git-trend.html")
	}
	if got == "" {
		t.Error("writeChart returned empty path")
	}
	content, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("temp HTML file not readable at %q: %v", got, err)
	}
	if len(content) == 0 {
		t.Error("temp HTML file is empty")
	}
}
