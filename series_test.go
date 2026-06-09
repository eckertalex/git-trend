package main

import (
	"fmt"
	"testing"
	"time"
)

func day(d int) time.Time {
	return time.Date(2024, 1, d, 12, 0, 0, 0, time.UTC)
}

func fixture() []Commit {
	return []Commit{
		{Hash: "1", AuthorName: "Alex Eckert", AuthorEmail: "alex@example.com", When: day(1)},
		{Hash: "2", AuthorName: "Sam Tester", AuthorEmail: "sam@example.com", When: day(2)},
		{Hash: "3", AuthorName: "Alex Eckert", AuthorEmail: "alex@example.com", When: day(3)},
		{Hash: "4", AuthorName: "Sam Tester", AuthorEmail: "sam@example.com", When: day(4)},
	}
}

func TestBuildWholeRepoCumulative(t *testing.T) {
	chart, err := Build(fixture(), BuildOptions{ShowTotal: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(chart.Series))
	}
	if chart.Series[0].Label != "total" {
		t.Errorf("label = %q, want %q", chart.Series[0].Label, "total")
	}
	want := []float64{0, 2, 3, 4}
	if !equal(chart.Series[0].Y, want) {
		t.Errorf("Y = %v, want %v", chart.Series[0].Y, want)
	}
	if !chart.Start.Equal(day(1)) || !chart.End.Equal(day(4)) {
		t.Errorf("range = %v..%v, want %v..%v", chart.Start, chart.End, day(1), day(4))
	}
	if last := chart.Series[0].Y[len(chart.Series[0].Y)-1]; last != 4 {
		t.Errorf("final cumulative = %v, want 4", last)
	}
}

func TestBuildAuthorsSubstringMatch(t *testing.T) {
	chart, err := Build(fixture(), BuildOptions{Authors: []string{"alex", "sam"}, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(chart.Series))
	}

	// Alex: commits at day(1) and day(3).
	wantAlex := []float64{0, 1, 2, 2}
	if chart.Series[0].Label != "Alex Eckert <alex@example.com>" || !equal(chart.Series[0].Y, wantAlex) {
		t.Errorf("alex series = %+v, want Y=%v", chart.Series[0], wantAlex)
	}

	// Sam: commits at day(2) and day(4).
	wantSam := []float64{0, 1, 1, 2}
	if chart.Series[1].Label != "Sam Tester <sam@example.com>" || !equal(chart.Series[1].Y, wantSam) {
		t.Errorf("sam series = %+v, want Y=%v", chart.Series[1], wantSam)
	}
}

// 4 authors on @frontend.dev, 5 on @backend.dev; "dev" matches all 9.
func manyAuthors() []Commit {
	var commits []Commit
	add := func(i int, team string) {
		commits = append(commits, Commit{
			Hash:        fmt.Sprint(i),
			AuthorName:  fmt.Sprintf("%s Dev %02d", team, i),
			AuthorEmail: fmt.Sprintf("dev%02d@%s.dev", i, team),
			When:        day(i),
		})
	}
	for i := 1; i <= 4; i++ {
		add(i, "frontend")
	}
	for i := 5; i <= 9; i++ {
		add(i, "backend")
	}
	return commits
}

func TestBuildFansOutToOneLinePerAuthor(t *testing.T) {
	// "frontend" matches 4 distinct authors, all within the cap.
	chart, err := Build(manyAuthors(), BuildOptions{Authors: []string{"frontend"}, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 4 {
		t.Fatalf("expected 4 author lines, got %d", len(chart.Series))
	}
}

func TestBuildTooManyAuthorsSingleTerm(t *testing.T) {
	// "dev" matches all 9 authors; the cap keeps the top 8 by commit count.
	chart, err := Build(manyAuthors(), BuildOptions{Authors: []string{"dev"}, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != maxSeries {
		t.Fatalf("expected %d series, got %d", maxSeries, len(chart.Series))
	}
	if chart.Dropped.Authors != 1 {
		t.Errorf("dropped.Authors = %d, want 1", chart.Dropped.Authors)
	}
}

func TestBuildTooManyAuthorsCombined(t *testing.T) {
	// 4 + 5 = 9 lines across two terms; the cap keeps the top 8 by commit count.
	chart, err := Build(manyAuthors(), BuildOptions{Authors: []string{"frontend", "backend"}, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != maxSeries {
		t.Fatalf("expected %d series, got %d", maxSeries, len(chart.Series))
	}
	if chart.Dropped.Authors != 1 {
		t.Errorf("dropped.Authors = %d, want 1", chart.Dropped.Authors)
	}
}

func TestBuildEmpty(t *testing.T) {
	chart, err := Build(nil, BuildOptions{Width: 10})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if chart.Series != nil {
		t.Errorf("expected nil series, got %v", chart.Series)
	}
	if !chart.Start.IsZero() || !chart.End.IsZero() {
		t.Errorf("expected zero times, got %v..%v", chart.Start, chart.End)
	}
}

func TestBuildSampleCount(t *testing.T) {
	chart, err := Build(fixture(), BuildOptions{ShowTotal: true, Width: 20})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if got := len(chart.Series[0].Y); got != 20 {
		t.Errorf("sample count = %d, want 20", got)
	}
}

func TestBuildTopAutoFill(t *testing.T) {
	// fixture() has 2 authors (Alex 2 commits, Sam 2 commits). --top fills both.
	chart, err := Build(fixture(), BuildOptions{ShowTop: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 2 {
		t.Fatalf("expected 2 series, got %d", len(chart.Series))
	}
	// Both have equal commit counts; first-seen order (date ascending) puts Alex before Sam.
	if chart.Series[0].Label != "Alex Eckert <alex@example.com>" {
		t.Errorf("series[0].Label = %q, want Alex first", chart.Series[0].Label)
	}
	if chart.Series[1].Label != "Sam Tester <sam@example.com>" {
		t.Errorf("series[1].Label = %q, want Sam second", chart.Series[1].Label)
	}
}

func TestBuildTopCappedAt8(t *testing.T) {
	// manyAuthors() has 9 distinct authors; --top should return exactly 8.
	chart, err := Build(manyAuthors(), BuildOptions{ShowTop: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != maxSeries {
		t.Fatalf("expected %d series, got %d", maxSeries, len(chart.Series))
	}
}

func TestBuildAuthorAndTopDedup(t *testing.T) {
	// --author alex --top: Alex is shown once (via --author), Sam fills the top slot.
	chart, err := Build(fixture(), BuildOptions{Authors: []string{"alex"}, ShowTop: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 2 {
		t.Fatalf("expected 2 series (alex + sam top-fill), got %d", len(chart.Series))
	}
	if chart.Series[0].Label != "Alex Eckert <alex@example.com>" {
		t.Errorf("series[0].Label = %q, want Alex (explicit author)", chart.Series[0].Label)
	}
	if chart.Series[1].Label != "Sam Tester <sam@example.com>" {
		t.Errorf("series[1].Label = %q, want Sam (top-fill)", chart.Series[1].Label)
	}
}

func TestBuildTotalLine(t *testing.T) {
	chart, err := Build(fixture(), BuildOptions{ShowTotal: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 1 {
		t.Fatalf("expected 1 series, got %d", len(chart.Series))
	}
	if chart.Series[0].Label != "total" {
		t.Errorf("label = %q, want %q", chart.Series[0].Label, "total")
	}
}

func TestBuildTotalAndTop(t *testing.T) {
	// --total --top with 2 authors: total + Alex + Sam = 3 series.
	chart, err := Build(fixture(), BuildOptions{ShowTotal: true, ShowTop: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 3 {
		t.Fatalf("expected 3 series (all + 2 top), got %d", len(chart.Series))
	}
	if chart.Series[0].Label != "total" {
		t.Errorf("series[0].Label = %q, want \"total\"", chart.Series[0].Label)
	}
}

func TestBuildTotalAndTopCappedAt8(t *testing.T) {
	// --total --top with 9 authors: total takes 1 slot, top fills the remaining 7.
	chart, err := Build(manyAuthors(), BuildOptions{ShowTotal: true, ShowTop: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != maxSeries {
		t.Fatalf("expected %d series, got %d", maxSeries, len(chart.Series))
	}
	if chart.Series[0].Label != "total" {
		t.Errorf("series[0].Label = %q, want \"total\"", chart.Series[0].Label)
	}
}

func TestBuildAuthorsBeatTotalForSlots(t *testing.T) {
	// "dev" matches 9 authors; explicit authors win all 8 slots, so --total is dropped.
	chart, err := Build(manyAuthors(), BuildOptions{ShowTotal: true, Authors: []string{"dev"}, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != maxSeries {
		t.Fatalf("expected %d series, got %d", maxSeries, len(chart.Series))
	}
	if !chart.Dropped.Total {
		t.Error("expected dropped.Total = true (no slot left for --total)")
	}
	for _, s := range chart.Series {
		if s.Label == "total" {
			t.Error("total line should have been dropped")
		}
	}
}

func TestBuildExplicitAuthorBeatsTopFill(t *testing.T) {
	var commits []Commit
	// alice: a single commit, the lowest count of anyone.
	commits = append(commits, Commit{Hash: "a", AuthorName: "Alice", AuthorEmail: "alice@x.com", When: day(1)})
	// 10 busy authors with 5 commits each — top-fill would pick these over alice.
	for b := range 10 {
		for c := range 5 {
			commits = append(commits, Commit{
				Hash:        fmt.Sprintf("b%d-%d", b, c),
				AuthorName:  fmt.Sprintf("Busy%02d", b),
				AuthorEmail: fmt.Sprintf("busy%02d@x.com", b),
				When:        day(2 + c),
			})
		}
	}

	chart, err := Build(commits, BuildOptions{Authors: []string{"alice"}, ShowTop: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != maxSeries {
		t.Fatalf("expected %d series, got %d", maxSeries, len(chart.Series))
	}
	found := false
	for _, s := range chart.Series {
		if s.Label == "Alice <alice@x.com>" {
			found = true
		}
	}
	if !found {
		t.Error("explicit --author alice should survive top-fill despite the fewest commits")
	}
}

func TestFitAuthorsCapsAndReports(t *testing.T) {
	// "dev" matches all 9 authors; fitAuthors keeps maxSeries and reports the rest.
	kept, dropped := fitAuthors([]string{"dev"}, manyAuthors(), maxSeries)
	if len(kept) != maxSeries {
		t.Errorf("kept = %d, want %d", len(kept), maxSeries)
	}
	if dropped != 1 {
		t.Errorf("dropped = %d, want 1", dropped)
	}
}

func TestBuildDistinctIdentitiesSharedEmail(t *testing.T) {
	// "Alex" and "Alexander" share an email but are distinct git identities.
	commits := []Commit{
		{Hash: "1", AuthorName: "Alex", AuthorEmail: "a@x.com", When: day(1)},
		{Hash: "2", AuthorName: "Alexander", AuthorEmail: "a@x.com", When: day(2)},
	}
	chart, err := Build(commits, BuildOptions{ShowTop: true, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 2 {
		t.Fatalf("expected 2 series (one per identity), got %d", len(chart.Series))
	}
	labels := map[string]bool{chart.Series[0].Label: true, chart.Series[1].Label: true}
	if !labels["Alex <a@x.com>"] || !labels["Alexander <a@x.com>"] {
		t.Errorf("unexpected labels: %v", []string{chart.Series[0].Label, chart.Series[1].Label})
	}
}

func TestBuildOverlappingPatternsDeduplicateByIdentity(t *testing.T) {
	// Three distinct identities; two overlapping patterns each match all three.
	// Expect exactly 3 lines, not 6.
	commits := []Commit{
		{Hash: "1", AuthorName: "Alex", AuthorEmail: "alex@x.com", When: day(1)},
		{Hash: "2", AuthorName: "Alexander", AuthorEmail: "alex@x.com", When: day(2)},
		{Hash: "3", AuthorName: "Alex", AuthorEmail: "alexander@x.com", When: day(3)},
	}
	chart, err := Build(commits, BuildOptions{Authors: []string{"alex", "lex"}, Width: 4})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if len(chart.Series) != 3 {
		t.Fatalf("expected 3 series (one per identity), got %d", len(chart.Series))
	}
}

func equal(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
