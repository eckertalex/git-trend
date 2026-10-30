# git-trend

Plots git commit history by author in the terminal. Run `git trend` in any
repo, add authors, set a date range, and the chart updates as you go.

The binary is named `git-trend`, so git picks it up as `git trend` once it's
on your `PATH`.

## TUI

![TUI with multiple authors](assets/git-trend-tui.png)

Launch it with no arguments in any git repo:

```
git trend
```

| Key          | Action                                                           |
| ------------ | ---------------------------------------------------------------- |
| `a`          | Add an author (name/email substring)                             |
| `d`          | Remove the selected author / clear the selected time range row   |
| `D`          | Clear all authors                                                |
| `t`          | Fill remaining slots with the top contributors                   |
| `m`          | Add yourself (`git config user.email`)                           |
| `T`          | Toggle the cumulative total line                                 |
| `s`          | Edit the since date                                              |
| `u`          | Edit the until date                                              |
| `j / k`      | Move down / up within the focused panel                          |
| `h / l`      | Switch focus between panels                                      |
| `Enter`      | Edit the focused time range row; confirm text input              |
| `w`          | Open the chart in the browser (writes to a temp file in $TMPDIR) |
| `e`          | Export the chart to `git-trend.html`                             |
| `?`          | Show all keybindings                                             |
| `q / Ctrl+C` | Quit                                                             |

Up to 8 author lines can be shown at once (one per palette color).

## CLI flags

Pass flags to pre-load the TUI or skip it entirely.

```sh
git trend                                          # open TUI
git trend --author alice                           # TUI pre-loaded with alice
git trend --top                                    # TUI pre-loaded with top contributors
git trend --total                                  # TUI pre-loaded with the total line
git trend --since 2024-01-01 --until 2025-01-01    # fixed window
git trend --web                                    # interactive HTML chart in browser
git trend --export chart.html                      # write HTML chart to file
```

| Flag        | Description                                                                       |
| ----------- | --------------------------------------------------------------------------------- |
| `--author`  | Author name/email substring to plot as its own line (repeatable).                 |
| `--top`     | Pre-load the top contributors by commit count.                                    |
| `--total`   | Include a cumulative line totaling every commit (counts toward the 8-line cap).   |
| `--me`      | Add yourself as an author line.                                                   |
| `--since`   | Only count commits after this date (passed verbatim to git).                      |
| `--until`   | Only count commits before this date.                                              |
| `--web`     | Open an interactive HTML chart in the browser (writes to a temp file in $TMPDIR). |
| `--export`  | Write the HTML chart to PATH (use `--web` to also open it).                       |

`git trend help` prints the usage and `git trend version` the version.

`--since` and `--until` accept any date git understands: `"2 weeks ago"`,
`"2024-01-01"`, `"last month"`, etc.

## Web output

![HTML chart in browser](assets/git-trend-web-chart.png)

```sh
git trend --web
git trend --export report.html --author alice --author bob
```

## Build

`build.sh` builds into `build/`.

```sh
./build.sh                  # debug, build/git-trend
./build.sh release          # optimized and stripped
./build/git-trend --web      # run what you built
./build.sh test
```

Needs: Go from `.tool-versions`.

## Install

```sh
./build.sh release
cp build/git-trend ~/.local/bin/
```

`~/.local/bin` needs to be on your `PATH`. To uninstall, delete the binary.
