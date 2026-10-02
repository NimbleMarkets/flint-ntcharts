package tui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/charmbracelet/x/ansi"
)

// Compiler abstracts the flint compiler: native (embedded wasm via
// compile.Runner) or browser (booba-shim flintchart via an app adapter).
// Implementations must apply opts to the input (envelope.Apply) before
// compiling; dropping them breaks fit-to-window silently.
type Compiler interface {
	Compile(ctx context.Context, input []byte, opts ...envelope.Option) (spec.Spec, []envelope.Warning, error)
}

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
	compiler Compiler

	width, height int
	raw           []byte // last received document (re-rendered on resize)
	source        string

	pane     Pane
	warnings []envelope.Warning
	err      error
	waiting  bool   // no document received yet
	notice   string // one-off message from a key (g), shown on the status line until the next render

	// gen is bumped on every InputMsg/WindowSizeMsg so in-flight renderedMsg
	// results from a superseded request can be told apart from the latest
	// one and dropped (renders run concurrently and can complete out of order).
	gen int
}

// New builds a Model backed by c: a native *compile.Runner (embedded wasm) or
// any other Compiler implementation (e.g. a browser/js host adapter).
func New(c Compiler) Model {
	return Model{compiler: c, waiting: true, pane: NewPane()}
}

func (m Model) Init() tea.Cmd { return m.pane.Init() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.gen++
		return m, tea.Batch(m.pane.SetSize(m.width, max(m.height-1, 0)), m.rerenderCmd())
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
		m.notice = ""
		if msg.err == nil {
			// errors keep the last good chart
			return m, m.pane.Apply(Frame{Text: msg.view, Image: msg.img})
		}
		return m, nil
	case SourceErrMsg:
		m.err = msg.Err
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "g":
			cmd, note := m.pane.Toggle() // Kitty graphics <-> glyphs, for raster charts
			m.notice = note
			return m, cmd
		}
	}
	// Anything else is the pane's: terminal probe replies, Kitty frames.
	return m, m.pane.Update(msg)
}

// rerenderCmd re-renders the retained document asynchronously (compiles run
// ~22ms; keep the update loop responsive).
func (m Model) rerenderCmd() tea.Cmd {
	if m.raw == nil || m.width <= 0 || m.height <= 1 {
		return nil
	}
	compiler, raw, gen := m.compiler, m.raw, m.gen
	w, h := m.width, m.height-1 // reserve the status line
	cellW, cellH := m.pane.CellPixelSize()
	return func() tea.Msg {
		msg := renderDoc(compiler, raw, w, h, cellW, cellH)
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
		body = m.pane.View()
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
	case m.notice != "":
		n := m.width - lipgloss.Width(left) - 2
		if n <= 0 {
			n = 0
		}
		line = left + "  " + statusStyle.Render(ansi.Truncate(m.notice, n, "…"))
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
