package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// sets up a throwaway git repo with a few commits, returns its path.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	gitCmd := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append([]string{
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null",
		}, env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	gitCmd(nil, "init", "-q")

	commit := func(name, email, date, msg string) {
		t.Helper()
		if err := exec.Command("touch", dir+"/"+msg).Run(); err != nil {
			t.Fatalf("touch: %v", err)
		}
		gitCmd(nil, "add", "-A")
		env := []string{
			"GIT_AUTHOR_NAME=" + name,
			"GIT_AUTHOR_EMAIL=" + email,
			"GIT_AUTHOR_DATE=" + date,
			"GIT_COMMITTER_NAME=" + name,
			"GIT_COMMITTER_EMAIL=" + email,
			"GIT_COMMITTER_DATE=" + date,
		}
		gitCmd(env, "commit", "-q", "-m", msg)
	}

	commit("Alice", "alice@example.com", "2024-01-01T12:00:00", "c1")
	commit("Bob", "bob@example.com", "2024-02-01T12:00:00", "c2")
	commit("Alice", "alice@example.com", "2024-03-01T12:00:00", "c3")

	return dir
}

func TestWriteChartNoCommits(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	cmd.Env = []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	t.Chdir(dir)

	_, err := writeChart(BuildOptions{ShowTotal: true}, "", "", filepath.Join(dir, "chart.html"))
	if err == nil {
		t.Fatal("expected error for repo with no commits, got nil")
	}
	if !strings.Contains(err.Error(), "commit") {
		t.Errorf("error = %q, want it to mention commits", err.Error())
	}
}
