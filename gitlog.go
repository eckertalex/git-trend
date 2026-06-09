package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// unit separator (0x1F) — can't appear in metadata, so names with spaces/commas parse safely
const fieldSep = "\x1f"

type Commit struct {
	Hash        string
	AuthorName  string
	AuthorEmail string
	When        time.Time
}

type Options struct {
	Since string
	Until string
}

func Log(opts Options) ([]Commit, error) {
	args := []string{
		"log",
		"--no-merges",
		"--pretty=format:%H" + fieldSep + "%aN" + fieldSep + "%aE" + fieldSep + "%cI",
	}

	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}

	cmd := exec.Command("git", args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	commits, parseErr := parseCommits(stdout)
	waitErr := cmd.Wait()

	if parseErr != nil {
		return nil, parseErr
	}

	if waitErr != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("git log: %s", msg)
		}

		return nil, fmt.Errorf("git log: %w", waitErr)
	}

	return commits, nil
}

// repoName returns the base directory name of the current git repository.
// Returns an empty string if the repo root cannot be determined.
//
// In a bare-repo + worktree setup (e.g. using git-worktree with a bare clone),
// --show-toplevel returns the worktree dir, not the repo root. --git-common-dir
// points to the shared bare repo directory, whose parent is the repo root.
// For a normal main worktree, --git-common-dir is the relative string ".git",
// so we fall back to --show-toplevel in that case.
func repoName() string {
	commonDir, err := exec.Command("git", "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return ""
	}
	dir := strings.TrimSpace(string(commonDir))
	if filepath.IsAbs(dir) {
		// Linked worktree or bare-repo worktree: repo root is the parent of
		// the common git dir (e.g. /path/to/repo/.bare -> repo).
		return filepath.Base(filepath.Dir(dir))
	}
	// Main worktree: --git-common-dir is relative (".git"), use --show-toplevel.
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return ""
	}
	return filepath.Base(strings.TrimSpace(string(top)))
}

// CurrentGitUser returns the identity string "Name <email>" from git config.
// Returns an error if either user.name or user.email is unset.
func CurrentGitUser() (identity string, err error) {
	nameOut, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return "", fmt.Errorf("git config user.name: %w", err)
	}

	emailOut, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		return "", fmt.Errorf("git config user.email: %w", err)
	}

	name := strings.TrimSpace(string(nameOut))
	email := strings.TrimSpace(string(emailOut))
	if name == "" || email == "" {
		return "", fmt.Errorf("git config user.name or user.email is not set")
	}

	return fmt.Sprintf("%s <%s>", name, email), nil
}

func parseCommits(r io.Reader) ([]Commit, error) {
	var commits []Commit

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		c, err := parseLine(line)
		if err != nil {
			return nil, err
		}

		commits = append(commits, c)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	return commits, nil
}

func parseLine(line string) (Commit, error) {
	fields := strings.Split(line, fieldSep)
	if len(fields) != 4 {
		return Commit{}, fmt.Errorf("unexpected git log line: %q", line)
	}

	when, err := time.Parse(time.RFC3339, fields[3])
	if err != nil {
		return Commit{}, fmt.Errorf("parsing commit date %q: %w", fields[3], err)
	}

	return Commit{
		Hash:        fields[0],
		AuthorName:  fields[1],
		AuthorEmail: fields[2],
		When:        when,
	}, nil
}
