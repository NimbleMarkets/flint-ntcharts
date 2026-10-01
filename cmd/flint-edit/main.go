// Command flint-edit is a live split-pane playground: edit a Flint chart spec
// (or a raw ntcharts-spec) with syntax highlighting in the left pane and watch
// it compile and render in the right pane on every keystroke. It shares the
// exact compile→build→render path used by flint-tui, so what you see here is
// what the renderer produces.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ionut-t/goeditor"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/flint-ntcharts/tui"
)

// highlightTheme is the Chroma style used for JSON token colors. Any style in
// github.com/alecthomas/chroma/v2/styles works (e.g. "github" for light
// terminals, "monokai", "nord").
const highlightTheme = "catppuccin-mocha"

var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true) // cyan
	hintStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))            // grey
	divStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	warnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	errStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red
)

// renderedMsg carries an async render result back into Update, tagged with the
// generation that requested it so stale results can be dropped.
type renderedMsg struct {
	gen   int
	view  string
	warns []envelope.Warning
	err   error
}

type model struct {
	runner tui.Compiler
	ed     goeditor.Model

	w, h  int
	src   string // last content sent to a render
	chart string // last good chart
	msg   string // status line text
	state int    // 0 pending, 1 ok, 2 warn, 3 error
	gen   int
	ex    int // index of the example last loaded; ctrl+n continues from it
}

func newModel(c tui.Compiler) model {
	ed := goeditor.New(40, 20)
	ed.DisableVimMode(true) // plain, non-modal typing
	ed.SetLanguage("json", highlightTheme)
	ed.SetContent(examples[0].src)
	_ = ed.SetCursorPositionEnd()
	ed.Focus()
	return model{runner: c, ed: ed, src: examples[0].src, msg: "type to render"}
}

func (m model) Init() tea.Cmd { return m.ed.Init() }

func (m *model) resize() {
	m.ed.SetSize(m.editorWidth(), m.contentH())
}

func (m model) editorWidth() int {
	w := m.w * 2 / 5
	if w < 30 {
		w = 30
	}
	if w > m.w-24 {
		w = m.w - 24
	}
	if w < 12 {
		w = 12
	}
	return w
}

func (m model) contentH() int {
	h := m.h - 2 // title row + status row
	if h < 1 {
		h = 1
	}
	return h
}

// chartArea is the cell box the chart renders into: everything right of the
// editor and its divider, over the content height.
func (m model) chartArea() (int, int) {
	w := m.w - m.editorWidth() - 3
	if w < 4 {
		w = 4
	}
	return w, m.contentH()
}

func (m model) renderCmd() tea.Cmd {
	if m.w <= 0 || m.h <= 2 {
		return nil
	}
	w, h := m.chartArea()
	runner, src, gen := m.runner, m.src, m.gen
	return func() tea.Msg {
		view, warns, err := tui.Render(runner, []byte(src), w, h)
		return renderedMsg{gen: gen, view: view, warns: warns, err: err}
	}
}

// exampleChord maps a bubbletea key string ("ctrl+1", "alt+3", ...) to a
// 0-based example index. Both modifiers are bound because ctrl+digit only
// reaches the program in kitty-protocol terminals; alt+digit works nearly
// everywhere and is the portable fallback.
func exampleChord(key string) (int, bool) {
	var digit string
	switch {
	case strings.HasPrefix(key, "ctrl+"):
		digit = strings.TrimPrefix(key, "ctrl+")
	case strings.HasPrefix(key, "alt+"):
		digit = strings.TrimPrefix(key, "alt+")
	default:
		return 0, false
	}
	if len(digit) != 1 || digit[0] < '1' || digit[0] > '9' {
		return 0, false
	}
	return int(digit[0] - '1'), true
}

// loadExample replaces the editor buffer with examples[i] and kicks off a
// render. Current edits are discarded — flint-edit is a playground.
func (m *model) loadExample(i int) tea.Cmd {
	ex := examples[i]
	m.ex = i
	m.ed.SetContent(ex.src)
	_ = m.ed.SetCursorPositionEnd()
	// SetContent only swaps the buffer; the editor repaints its viewport at
	// the end of Update. Without this pass the pane keeps showing the previous
	// example until the next keystroke reaches the editor.
	var edCmd tea.Cmd
	m.ed, edCmd = m.ed.Update(nil)
	m.src = ex.src
	m.gen++
	m.msg, m.state = "loaded example: "+ex.name, 0
	return tea.Batch(edCmd, m.renderCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.resize()
		// The editor lays out and syntax-highlights during Update, not on
		// SetSize alone — forward the tick so the pane paints highlighted.
		var edCmd tea.Cmd
		m.ed, edCmd = m.ed.Update(msg)
		m.gen++
		return m, tea.Batch(edCmd, m.renderCmd())

	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if i, ok := exampleChord(key); ok && i < len(examples) {
			return m, m.loadExample(i)
		}
		// ctrl+n steps to the next example, wrapping. Unlike the digit chords
		// it is a plain control byte, so it reaches the program in every
		// terminal (and in scripted recordings).
		if key == "ctrl+n" {
			return m, m.loadExample((m.ex + 1) % len(examples))
		}
		// chords past the example list (alt+9, ctrl+9) fall through to the editor,
		// as do legacy-terminal artifacts for unsupported ctrl+digit (e.g. ctrl+@)

	case renderedMsg:
		if msg.gen != m.gen {
			return m, nil // a newer edit already superseded this render
		}
		if msg.err != nil {
			m.msg, m.state = msg.err.Error(), 3 // keep the last good chart on screen
			return m, nil
		}
		m.chart = msg.view
		if len(msg.warns) > 0 {
			m.msg, m.state = fmt.Sprintf("%d warning(s): %s", len(msg.warns), msg.warns[0].Message), 2
		} else {
			m.msg, m.state = "compiled ok", 1
		}
		return m, nil
	}

	// everything else (typing, arrows, paste, cursor blink) drives the editor
	var cmd tea.Cmd
	m.ed, cmd = m.ed.Update(msg)
	if v := m.ed.GetCurrentContent(); v != m.src {
		m.src = v
		m.gen++
		return m, tea.Batch(cmd, m.renderCmd())
	}
	return m, cmd
}

func (m model) View() tea.View {
	title := titleStyle.Render("flint-edit") +
		hintStyle.Render("   edit the spec, watch it render  ·  ctrl+n next example (or ctrl/alt+1·2·3)  ·  ctrl+c quit")

	chart := m.chart
	if strings.TrimSpace(chart) == "" {
		chart = hintStyle.Render("(compiling…)")
	}

	h := m.contentH()
	divider := divStyle.Render(strings.TrimRight(strings.Repeat("│\n", h), "\n"))

	body := lipgloss.JoinHorizontal(lipgloss.Top, m.ed.View(), " ", divider, " ", chart)

	content := title + "\n" + body + "\n" + m.statusLine()
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m model) statusLine() string {
	style := hintStyle
	switch m.state {
	case 1:
		style = okStyle
	case 2:
		style = warnStyle
	case 3:
		style = errStyle
	}
	return style.Render(truncate(m.msg, m.w))
}

func truncate(s string, n int) string {
	if n <= 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner, err := compile.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "flint-edit: compiler init: %v\n", err)
		os.Exit(1)
	}
	defer runner.Close(ctx)

	p := tea.NewProgram(newModel(runner), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "flint-edit: %v\n", err)
		os.Exit(1)
	}
}
