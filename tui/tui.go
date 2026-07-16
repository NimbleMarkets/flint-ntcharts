package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/charmbracelet/x/ansi"
)

// InputMsg delivers a new JSON document from any source (file, stdin, socket).
type InputMsg struct {
	Raw    []byte
	Source string
}

// SourceErrMsg reports a source-level failure (read error, accept error).
type SourceErrMsg struct{ Err error }

var (
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// Model is the flint-tui bubbletea model. Zero value is not usable; use New.
type Model struct {
	runner *compile.Runner

	width, height int
	raw           []byte // last received document (re-rendered on resize)
	source        string

	chartView string
	warnings  []compile.Warning
	err       error
	waiting   bool // no document received yet

	// gen is bumped on every InputMsg/WindowSizeMsg so in-flight renderedMsg
	// results from a superseded request can be told apart from the latest
	// one and dropped (renders run concurrently and can complete out of order).
	gen int
}

func New(runner *compile.Runner) Model {
	return Model{runner: runner, waiting: true}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.gen++
		return m, m.rerenderCmd()
	case InputMsg:
		m.raw, m.source = msg.Raw, msg.Source
		m.waiting = false
		m.gen++
		return m, m.rerenderCmd()
	case renderedMsg:
		if msg.gen != m.gen {
			return m, nil // stale result from a superseded request
		}
		m.warnings, m.err = msg.warnings, msg.err
		if msg.err == nil {
			m.chartView = msg.view // errors keep the last good chart
		}
		return m, nil
	case SourceErrMsg:
		m.err = msg.Err
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// rerenderCmd re-renders the retained document asynchronously (compiles run
// ~22ms; keep the update loop responsive).
func (m Model) rerenderCmd() tea.Cmd {
	if m.raw == nil || m.width <= 0 || m.height <= 1 {
		return nil
	}
	runner, raw, gen := m.runner, m.raw, m.gen
	w, h := m.width, m.height-1 // reserve the status line
	return func() tea.Msg {
		msg := renderDoc(runner, raw, w, h)
		msg.gen = gen
		return msg
	}
}

func (m Model) View() tea.View {
	var body string
	switch {
	case m.waiting:
		body = statusStyle.Render("waiting for a chart document…")
	default:
		body = m.chartView
	}
	// pad/truncate body to exactly height-1 lines so the status line stays put
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	max := m.height - 1
	if max < 0 {
		max = 0
	}
	if len(lines) > max {
		lines = lines[:max]
	}
	for len(lines) < max {
		lines = append(lines, "")
	}
	v := tea.NewView(strings.Join(lines, "\n") + "\n" + m.statusLine())
	v.AltScreen = true
	return v
}

func (m Model) statusLine() string {
	left := statusStyle.Render(fmt.Sprintf("flint-tui · %s · %dx%d", orDash(m.source), m.width, m.height))
	var line string
	switch {
	case m.err != nil:
		n := m.width - lipgloss.Width(left) - 2
		if n <= 0 {
			n = 0
		}
		line = left + "  " + errStyle.Render(ansi.Truncate(m.err.Error(), n, "…"))
	case len(m.warnings) > 0:
		w := fmt.Sprintf("%d warning(s): %s", len(m.warnings), m.warnings[0].Message)
		n := m.width - lipgloss.Width(left) - 2
		if n <= 0 {
			n = 0
		}
		line = left + "  " + warnStyle.Render(ansi.Truncate(w, n, "…"))
	default:
		line = left
	}
	// Safety net: the suffix budgeting above sizes the error/warning text to
	// fit, but "left" itself (e.g. a long source path) can already overflow
	// m.width, which would wrap the status line. Clamp the whole line.
	return ansi.Truncate(line, m.width, "")
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
