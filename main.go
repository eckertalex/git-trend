package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/term"
)

var Version = "v0.0.0"

type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}

func openInBrowser(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

type cliFlags struct {
	since     string
	until     string
	showTotal bool
	showTop   bool
	web       bool
	export    string
	me        bool
	authors   stringList
}

func main() {
	var f cliFlags
	fs := flag.NewFlagSet("git-trend", flag.ExitOnError)
	showVersion := fs.Bool("version", false, "print the version and exit")
	fs.BoolVar(&f.me, "me", false, "include the current git user as a separate line")
	fs.BoolVar(&f.showTotal, "total", false, "include a cumulative line for total commits")
	fs.BoolVar(&f.showTop, "top", false, "plot the top contributors")
	fs.BoolVar(&f.web, "web", false, "open an interactive HTML chart in the browser")
	fs.StringVar(&f.export, "export", "", "write the HTML chart to PATH (use --web to also open it)")
	fs.StringVar(&f.since, "since", "", "only include commits after this date (e.g. \"2 weeks ago\")")
	fs.StringVar(&f.until, "until", "", "only include commits before this date")
	fs.Var(&f.authors, "author", "author name/email substring to plot (repeatable)")

	// Short aliases mirror the in-TUI keybindings.
	fs.Var(&f.authors, "a", "shorthand for --author")
	fs.BoolVar(&f.me, "m", false, "shorthand for --me")
	fs.BoolVar(&f.showTotal, "T", false, "shorthand for --total")
	fs.BoolVar(&f.showTop, "t", false, "shorthand for --top")
	fs.BoolVar(&f.web, "w", false, "shorthand for --web")
	fs.StringVar(&f.export, "e", "", "shorthand for --export")
	fs.StringVar(&f.since, "s", "", "shorthand for --since")
	fs.StringVar(&f.until, "u", "", "shorthand for --until")
	fs.Parse(os.Args[1:])

	if *showVersion {
		fmt.Println("git-trend", Version)
		return
	}

	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		fmt.Fprintln(os.Stderr, "git-trend: fatal: not a git repository")
		os.Exit(1)
	}

	if err := run(f); err != nil {
		fmt.Fprintln(os.Stderr, "git-trend: fatal:", err)
		os.Exit(1)
	}
}

func run(f cliFlags) error {
	authors := f.authors

	if f.me {
		identity, err := CurrentGitUser()
		if err != nil {
			return fmt.Errorf("--me: %w", err)
		}

		authors = append(authors, identityPattern(identity))
	}

	buildOpts := BuildOptions{
		Authors:   authors,
		ShowTop:   f.showTop,
		ShowTotal: f.showTotal || (len(authors) == 0 && !f.showTop),
	}

	if f.export != "" || f.web {
		path, err := writeChart(buildOpts, f.since, f.until, f.export)
		if err != nil {
			return err
		}

		if f.web {
			return openInBrowser(path)
		}

		return nil
	}

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("stdout is not a terminal (use --web for a browser chart, or --export=FILE to save one)")
	}

	return RunTUI(buildOpts, f.since, f.until)
}
