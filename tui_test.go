package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// testCommits returns a small deterministic commit set for driving TUI tests.
// No git repo or mocks involved -- real Commit structs, real Update paths.
func testCommits() []Commit {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	commits := make([]Commit, 30)
	for i := range commits {
		day := base.AddDate(0, 0, i)
		name, email := "Alice Example", "alice@example.com"
		if i%3 == 0 {
			name, email = "Bob Example", "bob@example.com"
		}
		commits[i] = Commit{
			Hash:        "abc" + strings.Repeat("0", i%8),
			AuthorName:  name,
			AuthorEmail: email,
			When:        day,
		}
	}
	return commits
}

// newTestModel returns a model sized for tests and pre-populated with commits
// so it doesn't need to shell out to git.
func newTestModel() tuiModel {
	m := newTUIModel(BuildOptions{ShowTop: true}, "", "")
	m.width = 120
	m.height = 40
	m.authorsList.SetSize(leftPanelWidth-2, m.authorsPanelHeight())
	return m
}

// sendCommits delivers a commitsMsg to m and returns the updated model.
func sendCommits(m tuiModel, commits []Commit) tuiModel {
	updated, _ := m.Update(commitsMsg{commits: commits})
	return updated.(tuiModel)
}

func TestSpinnerShownWhileLoading(t *testing.T) {
	m := newTestModel()
	// Before commits arrive, loading is true.
	if !m.loading {
		t.Fatal("expected loading=true before commits arrive")
	}
	view := m.View()
	// The chart panel shows the spinner text.
	if !strings.Contains(view, "Fetching commits") {
		t.Errorf("expected 'Fetching commits' in loading view, got:\n%s", view)
	}
}

func TestAuthorsListAfterCommits(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())

	if m.loading {
		t.Fatal("expected loading=false after commits arrive")
	}

	view := m.View()
	// Both authors should appear somewhere in the view.
	if !strings.Contains(view, "Alice") {
		t.Errorf("expected 'Alice' in view after commits, got:\n%s", view)
	}
	if !strings.Contains(view, "Bob") {
		t.Errorf("expected 'Bob' in view after commits, got:\n%s", view)
	}
	// Commit counts should be shown as numbers next to each author.
	if !strings.Contains(view, "20") || !strings.Contains(view, "10") {
		t.Errorf("expected commit counts (20, 10) in view, got:\n%s", view)
	}
}

func TestFocusCycles(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())

	initial := m.focusedBox
	// Tab once -- should advance focus.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = updated.(tuiModel)
	if m.focusedBox == initial {
		t.Errorf("focus did not advance on 'l': still box %d", m.focusedBox)
	}
	// Tab numBoxes times total -- should wrap back.
	for range numBoxes - 1 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		m = updated.(tuiModel)
	}
	if m.focusedBox != initial {
		t.Errorf("expected focus to wrap back to %d after full cycle, got %d", initial, m.focusedBox)
	}
}

func TestAddAuthorMode(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())

	// Press 'a' to enter add-author mode.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(tuiModel)
	if m.mode != modeAddAuthor {
		t.Fatalf("expected modeAddAuthor after 'a', got %v", m.mode)
	}

	// Esc should return to normal.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	if m.mode != modeNormal {
		t.Fatalf("expected modeNormal after esc, got %v", m.mode)
	}
}

func TestHelpMenuOpenClose(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())

	// '?' opens the menu.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(tuiModel)
	if m.mode != modeMenu {
		t.Fatalf("expected modeMenu after '?', got %v", m.mode)
	}
	view := m.View()
	if !strings.Contains(view, "Keybindings") {
		t.Errorf("expected 'Keybindings' overlay in view, got:\n%s", view)
	}

	// Esc closes it.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	if m.mode != modeNormal {
		t.Fatalf("expected modeNormal after esc, got %v", m.mode)
	}
}

func TestAuthorDrillInAndBack(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())

	// Authors box is focused by default; enter drills into the selected author.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if m.rightView != viewAuthor {
		t.Fatalf("expected viewAuthor after enter, got %v", m.rightView)
	}

	view := m.View()
	if !strings.Contains(view, "Activity") {
		t.Errorf("expected 'Activity' section in profile, got:\n%s", view)
	}
	if !strings.Contains(view, "example.com") {
		t.Errorf("expected author email in profile, got:\n%s", view)
	}

	// Esc returns to the chart.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(tuiModel)
	if m.rightView != viewChart {
		t.Fatalf("expected viewChart after esc, got %v", m.rightView)
	}
	if m.detailAuthor != "" {
		t.Errorf("expected detailAuthor cleared after esc, got %q", m.detailAuthor)
	}
}

func TestAuthorDrillInSwitch(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	first := m.detailAuthor

	// 'j' moves the list selection; the profile follows without leaving the view.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(tuiModel)
	if m.rightView != viewAuthor {
		t.Fatalf("expected to stay in viewAuthor after 'j', got %v", m.rightView)
	}
	if m.detailAuthor == first {
		t.Errorf("expected detailAuthor to change after 'j', still %q", first)
	}
}

func TestAuthorViewAddOpensInput(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)

	// 'a' must still open the add-author input from within the author view.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(tuiModel)
	if m.mode != modeAddAuthor {
		t.Fatalf("expected modeAddAuthor after 'a' in author view, got %v", m.mode)
	}
}

func TestAuthorViewDeleteFollowsSelection(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	first := m.detailAuthor

	// 'd' removes the viewed author; the profile follows to the next selection.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	m = updated.(tuiModel)
	if m.rightView != viewAuthor {
		t.Fatalf("expected to remain in author view after delete, got %v", m.rightView)
	}
	if m.detailAuthor == first {
		t.Errorf("expected profile to move off the deleted author %q", first)
	}
}

func TestAuthorViewMenuClearExits(t *testing.T) {
	m := newTestModel()
	m = sendCommits(m, testCommits())
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)

	// Open the help menu from the author view.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	m = updated.(tuiModel)
	if m.mode != modeMenu {
		t.Fatalf("expected modeMenu after '?', got %v", m.mode)
	}

	// Navigate to "Clear all authors" (index 3) and select it. Routing through
	// handleAuthorKey must then exit the author view since no authors remain.
	for i := 0; i < 3; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(tuiModel)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if m.rightView != viewChart {
		t.Errorf("expected to exit author view after clearing authors via menu, got %v", m.rightView)
	}
}

func TestAuthorProfileIgnoresDateRange(t *testing.T) {
	m := newTestModel()
	// The chart sees only a narrow date-filtered slice...
	narrow := testCommits()[:6]
	m = sendCommits(m, narrow)
	// ...but the full history (no date filter) is the complete set.
	full := testCommits()
	updated, _ := m.Update(allCommitsMsg{commits: full})
	m = updated.(tuiModel)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)

	// Alice has 4 commits in the narrow window but 20 across full history; the
	// profile must report the full-history figure.
	if !strings.Contains(m.detailView, "20 (67% of repo)") {
		t.Errorf("expected full-history stats (20 of 30) in profile, got:\n%s", m.detailView)
	}
}

func TestAuthorProfileScroll(t *testing.T) {
	m := newTestModel()
	m.height = 12 // force the profile body to overflow the viewport
	m.authorsList.SetSize(leftPanelWidth-2, m.authorsPanelHeight())
	m = sendCommits(m, testCommits())

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(tuiModel)
	if m.authorVP.YOffset != 0 {
		t.Fatalf("expected fresh profile scrolled to top, got YOffset=%d", m.authorVP.YOffset)
	}

	// ctrl+d scrolls the profile down.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	m = updated.(tuiModel)
	if m.authorVP.YOffset == 0 {
		t.Fatalf("expected YOffset > 0 after ctrl+d")
	}

	// ctrl+u scrolls back up.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	m = updated.(tuiModel)
	if m.authorVP.YOffset != 0 {
		t.Errorf("expected YOffset back to 0 after ctrl+u, got %d", m.authorVP.YOffset)
	}
}

func TestQuitViaQ(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestModel())
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t)
}

func TestQuitViaCtrlC(t *testing.T) {
	tm := teatest.NewTestModel(t, newTestModel())
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t)
}
