package main

import (
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const leftPanelWidth = 24

const (
	boxAuthors    = 0
	boxTimeFrames = 1
	numBoxes      = 2
)

// rightView selects what the right panel shows: the aggregate chart or a single
// author's profile. The author view has its own key handler, so it is a distinct
// interaction context rather than a focus target.
type rightView int

const (
	viewChart rightView = iota
	viewAuthor
)

const (
	sinceHint = "e.g. 6 months ago · 2024-01-01"
	untilHint = "e.g. 2025-01-01 (empty = now)"
)

type mode int

const (
	modeNormal    mode = iota
	modeAddAuthor      // add-author text input active
	modeEditSince      // since date input active
	modeEditUntil      // until date input active
	modeMenu           // keybindings overlay active
)

type tuiModel struct {
	allCommits  []Commit
	series      []Series
	samples     []time.Time
	start, end  time.Time
	buildErr    error
	chartView   string
	initialized bool

	authors        []string
	showTotal      bool
	pendingTopFill bool
	since          string
	until          string

	fullCommits     []Commit            // unfiltered history; powers the author profile
	commitsByAuthor map[string][]Commit // fullCommits indexed by identity
	rightView       rightView
	detailAuthor    string // identity label of the drilled-in author
	detailView      string // cached rendered author-page body ("" = empty state)
	authorVP        viewport.Model

	repoName string

	colorByLabel map[string]int
	nextColor    int

	width, height int

	focusedBox int

	loading bool
	spinner spinner.Model

	mode        mode
	dateInput   textinput.Model
	authorInput textinput.Model
	statusMsg   string
	menuCursor  int

	authorsList   list.Model
	timeRangeList list.Model
}

func newTUIModel(opts BuildOptions, since, until string) tuiModel {
	ti := textinput.New()
	ti.Placeholder = "name or email (regex)"
	ti.CharLimit = 100

	di := textinput.New()
	di.CharLimit = 100

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))

	newList := func(items []list.Item, h int) list.Model {
		l := list.New(items, compactDelegate(), leftPanelWidth-2, h)
		l.SetShowTitle(false)
		l.SetShowHelp(false)
		l.SetShowStatusBar(false)
		l.SetShowPagination(false)
		l.SetFilteringEnabled(false)
		l.DisableQuitKeybindings()
		return l
	}

	// Compact delegate: 2 lines per item, no spacing between = 4 lines for 2 items.
	const trH = 4
	tr := newList([]list.Item{
		timeRangeItem{label: "since", value: since, placeholder: "all time"},
		timeRangeItem{label: "until", value: until, placeholder: "now"},
	}, trH)

	return tuiModel{
		authors:        append([]string{}, opts.Authors...),
		showTotal:      opts.ShowTotal,
		pendingTopFill: opts.ShowTop,
		since:          since,
		until:          until,
		authorInput:    ti,
		dateInput:      di,
		loading:        true,
		colorByLabel:   make(map[string]int),
		spinner:        sp,
		authorsList:    newList(nil, 3),
		timeRangeList:  tr,
		authorVP:       viewport.New(0, 0),
		repoName:       repoName(),
	}
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(fetchCommits(m.since, m.until), fetchAllCommits(), m.spinner.Tick)
}

func (m tuiModel) chartWidth() int {
	w := m.width - leftPanelWidth - 2
	if w < 10 {
		return 10
	}
	return w
}

func (m tuiModel) panelHeight() int {
	h := m.height - 1
	if h < 4 {
		return 4
	}
	return h
}

func (m tuiModel) authorsPanelHeight() int {
	const tfInnerH = 4 // 2-line compact delegate × 2 items, no spacing
	return max(m.panelHeight()-(tfInnerH+2)-1-2, 3)
}

func RunTUI(opts BuildOptions, since, until string) error {
	m := newTUIModel(opts, since, until)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
