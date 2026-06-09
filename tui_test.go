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
