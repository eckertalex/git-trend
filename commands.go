package main

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type commitsMsg struct {
	commits []Commit
	err     error
}

type currentUserMsg struct {
	identity string
	err      error
}

type webMsg struct{ err error }

type exportedMsg struct {
	path string
	err  error
}

func fetchCommits(since, until string) tea.Cmd {
	return func() tea.Msg {
		commits, err := Log(Options{Since: since, Until: until})
		return commitsMsg{commits, err}
	}
}

func fetchCurrentUser() tea.Cmd {
	return func() tea.Msg {
		identity, err := CurrentGitUser()
		return currentUserMsg{identity, err}
	}
}

func openWebFromTUI(s []Series, samples []time.Time, start, end time.Time) error {
	f, err := os.CreateTemp("", "git-trend-*.html")
	if err != nil {
		return err
	}
	path := f.Name()
	if err := RenderHTML(s, samples, start, end, repoName(), f); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return openInBrowser(path)
}

func exportWebFromTUI(s []Series, samples []time.Time, start, end time.Time) (string, error) {
	const path = "git-trend.html"
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	if err := RenderHTML(s, samples, start, end, repoName(), f); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	return path, nil
}
