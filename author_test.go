package main

import (
	"testing"
	"time"
)

// 2024-01-01 is a Monday, so day(d) maps directly to weekdays:
// day(1)=Mon day(2)=Tue day(3)=Wed day(4)=Thu day(5)=Fri day(6)=Sat day(7)=Sun.

func days(ds ...int) []time.Time {
	out := make([]time.Time, len(ds))
	for i, d := range ds {
		out[i] = dayOf(day(d))
	}
	return out
}

func TestLongestStreak(t *testing.T) {
	cases := []struct {
		name string
		in   []time.Time
		want int
	}{
		{"empty", nil, 0},
		{"single", days(1), 1},
		{"run then gap", days(1, 2, 3, 5), 3},
		{"all gaps", days(1, 3, 5), 1},
	}
	for _, tc := range cases {
		if got := topStreaks(calendarRuns(tc.in)).days; got != tc.want {
			t.Errorf("%s: longest streak = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestLongestStreakSpan(t *testing.T) {
	// Run from day 1..3 is the longest; its dates should be reported.
	got := topStreaks(calendarRuns(days(1, 2, 3, 5)))
	if got.days != 3 {
		t.Fatalf("days = %d, want 3", got.days)
	}
	if len(got.spans) != 1 {
		t.Fatalf("spans = %d, want 1", len(got.spans))
	}
	if !got.spans[0].start.Equal(dayOf(day(1))) || !got.spans[0].end.Equal(dayOf(day(3))) {
		t.Errorf("span = %v..%v, want %v..%v", got.spans[0].start, got.spans[0].end, dayOf(day(1)), dayOf(day(3)))
	}
}

func TestTopStreaksTies(t *testing.T) {
	// Two separate 2-day runs (1-2 and 5-6) tie for the record.
	got := topStreaks(calendarRuns(days(1, 2, 5, 6)))
	if got.days != 2 {
		t.Fatalf("days = %d, want 2", got.days)
	}
	if len(got.spans) != 2 {
		t.Errorf("spans = %d, want 2 tied runs", len(got.spans))
	}
}

func TestWeekdayStreak(t *testing.T) {
	cases := []struct {
		name string
		in   []time.Time
		want int
	}{
		// Fri(5) -> Mon(8): the gap is only Sat/Sun, so the run holds.
		{"weekend bridged", days(5, 8), 2},
		// Fri(5) -> Tue(9): Mon(8) is a missed weekday, so it breaks.
		{"missed weekday breaks", days(5, 9), 1},
		// Consecutive weekdays still count normally.
		{"plain run", days(1, 2, 3), 3},
	}
	for _, tc := range cases {
		if got := topStreaks(weekdayRuns(tc.in)).days; got != tc.want {
			t.Errorf("%s: weekday streak = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestHeatBucket(t *testing.T) {
	cases := map[int]int{0: 0, 1: 1, 2: 2, 3: 2, 4: 3, 5: 3, 6: 4, 99: 4}
	for n, want := range cases {
		if got := heatBucket(n); got != want {
			t.Errorf("heatBucket(%d) = %d, want %d", n, got, want)
		}
	}
}

func TestComputeAuthorStats(t *testing.T) {
	mk := func(d int) Commit {
		return Commit{AuthorName: "A", AuthorEmail: "a@x.com", When: day(d)}
	}
	// day(1) Mon x1, day(2) Tue x3, day(5) Fri x1 = 5 commits over 3 active days.
	commits := []Commit{mk(1), mk(2), mk(2), mk(2), mk(5)}

	s := computeAuthorStats(commits, len(commits))

	if s.total != 5 {
		t.Errorf("total = %d, want 5", s.total)
	}
	if s.pct != 100 {
		t.Errorf("pct = %v, want 100", s.pct)
	}
	if s.activeDays != 3 {
		t.Errorf("activeDays = %d, want 3", s.activeDays)
	}
	if !s.first.Equal(dayOf(day(1))) || !s.last.Equal(dayOf(day(5))) {
		t.Errorf("range = %v..%v, want %v..%v", s.first, s.last, dayOf(day(1)), dayOf(day(5)))
	}
	if !s.peakDay.Equal(dayOf(day(2))) || s.peakCount != 3 {
		t.Errorf("peak = %v (%d), want %v (3)", s.peakDay, s.peakCount, dayOf(day(2)))
	}
	// Tuesday has 3 commits on 1 active date.
	tue := s.weekdays[time.Tuesday]
	if tue.total != 3 || tue.activeDays != 1 || tue.peak != 3 {
		t.Errorf("Tuesday = %+v, want {total:3 activeDays:1 peak:3}", tue)
	}
	if s.dailyStreak.days != 2 { // Mon+Tue consecutive
		t.Errorf("daily streak = %d, want 2", s.dailyStreak.days)
	}
}
