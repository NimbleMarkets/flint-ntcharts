// Package editor is the model behind flint-edit, a live split-pane playground: edit a Flint chart spec
// (or a raw ntcharts-spec) with syntax highlighting in the left pane and watch
// it compile and render in the right pane on every keystroke. It shares the
// exact compile→build→render path used by flint-tui, so what you see here is
// what the renderer produces.
package editor

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/ionut-t/goeditor"

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
	frame tui.Frame
	err   error
}

type model struct {
	runner tui.Compiler
	ed     goeditor.Model

	w, h     int
	src      string   // last content sent to a render
	pane     tui.Pane // last good chart: text, or a raster image
	msg      string   // status line text
	state    int      // 0 pending, 1 ok, 2 warn, 3 error
	gen      int
	ex       int    // index of the example last loaded; ctrl+n continues from it
	renderer string // the ctrl+r override of the document's renderer: \"\", \"text\" or \"raster\"
}

// New returns the playground model, compiling through c. The same model runs
// in the terminal (cmd/flint-edit) and in the browser (demo/).
func New(c tui.Compiler) tea.Model {
	return newModel(c)
}

// NewAt is New starting on example i (0-based). An out-of-range i starts on the
// first example.
func NewAt(c tui.Compiler, i int) tea.Model {
	if i < 0 || i >= len(examples) {
		i = 0
	}
	m := newModel(c)
	m.ex = i
	m.ed.SetContent(examples[i].src)
	_ = m.ed.SetCursorPositionEnd()
	m.src = examples[i].src
	return m
}

// Count is the number of built-in examples.
func Count() int { return len(examples) }

func newModel(c tui.Compiler) model {
	ed := goeditor.New(40, 20)
	ed.DisableVimMode(true) // plain, non-modal typing
	ed.SetLanguage("json", highlightTheme)
	ed.SetContent(examples[0].src)
	_ = ed.SetCursorPositionEnd()
	ed.Focus()
	return model{runner: c, ed: ed, src: examples[0].src, msg: "type to render", pane: tui.NewPane()}
}

func (m model) Init() tea.Cmd { return tea.Batch(m.ed.Init(), m.pane.Init()) }

func (m *model) resize() tea.Cmd {
	m.ed.SetSize(m.editorWidth(), m.contentH())
	w, h := m.chartArea()
	return m.pane.SetSize(w, h)
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
	cellW, cellH := m.pane.CellPixelSize()
	opts := tui.FrameOptions{CellW: cellW, CellH: cellH, Renderer: m.renderer}
	return func() tea.Msg {
		frame, err := tui.RenderFrame(runner, []byte(src), w, h, opts)
		return renderedMsg{gen: gen, frame: frame, err: err}
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
	m.renderer = "" // the new example starts from its own renderer
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
		paneCmd := m.resize()
		// The editor lays out and syntax-highlights during Update, not on
		// SetSize alone — forward the tick so the pane paints highlighted.
		var edCmd tea.Cmd
		m.ed, edCmd = m.ed.Update(msg)
		m.gen++
		return m, tea.Batch(edCmd, paneCmd, m.renderCmd())

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
		// ctrl+r (alt+r in terminals that send Alt as Meta, for those where ctrl+r
		// is taken) flips the chart between the text and raster renderers. It is a view setting for
		// the example on screen, so it does not edit the document.
		if key == "ctrl+r" || key == "alt+r" {
			m.renderer = tui.ToggleRenderer([]byte(m.src), m.renderer)
			m.gen++
			return m, m.renderCmd()
		}
		// ctrl+g switches a raster chart between Kitty graphics and glyphs.
		if key == "ctrl+g" {
			cmd, note := m.pane.Toggle()
			m.msg, m.state = note, 0
			return m, cmd
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
		paneCmd := m.pane.Apply(msg.frame)
		if warns := msg.frame.Warnings; len(warns) > 0 {
			m.msg, m.state = fmt.Sprintf("%d warning(s): %s", len(warns), warns[0].Message), 2
		} else if msg.frame.Image != nil {
			m.msg, m.state = "compiled ok · raster", 1
		} else {
			m.msg, m.state = "compiled ok", 1
		}
		return m, paneCmd
	}

	// The pane's own messages (terminal probe replies, Kitty frames) are not
	// the editor's; give them to the pane first, then the editor as before.
	paneCmd := m.pane.Update(msg)

	// everything else (typing, arrows, paste, cursor blink) drives the editor
	var cmd tea.Cmd
	m.ed, cmd = m.ed.Update(msg)
	if v := m.ed.GetCurrentContent(); v != m.src {
		m.src = v
		m.gen++
		return m, tea.Batch(cmd, paneCmd, m.renderCmd())
	}
	return m, tea.Batch(cmd, paneCmd)
}

func (m model) View() tea.View {
	title := titleStyle.Render("flint-edit") +
		hintStyle.Render(fmt.Sprintf("   edit the spec, watch it render  ·  ctrl+n next example (or ctrl/alt+1-%d)  ·  ctrl+r renderer  ·  ctrl+g image mode  ·  ctrl+c quit", len(examples)))

	chart := m.pane.View()
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
	// The lower-right corner names the example and the kind of chart, so it is
	// clear what is on screen after switching or editing. The status message
	// gives way to it when the row is short.
	label := m.cornerLabel()
	labelW := ansi.StringWidth(label)
	if m.w <= labelW {
		return hintStyle.Render(ansi.Truncate(label, m.w, ""))
	}
	msg := ansi.Truncate(m.msg, m.w-labelW-1, "…")
	gap := m.w - ansi.StringWidth(msg) - labelW
	return style.Render(msg) + strings.Repeat(" ", gap) + hintStyle.Render(label)
}

// cornerLabel is "3/8 · Candlestick Chart": the example loaded last and the
// chart type in the spec now, so it follows edits. "(edited)" marks a spec that
// no longer matches the example it started as.
func (m model) cornerLabel() string {
	edited := ""
	if m.src != examples[m.ex].src {
		edited = " (edited)"
	}
	return fmt.Sprintf("%d/%d%s · %s", m.ex+1, len(examples), edited, chartKind(m.src))
}

// chartKind names the chart a document asks for: the flint chartType, else a
// raw ntcharts-spec's type (bare or inside a compile envelope).
func chartKind(src string) string {
	var doc struct {
		ChartSpec struct {
			ChartType string `json:"chartType"`
		} `json:"chart_spec"`
		Type string `json:"type"`
		Spec struct {
			Type string `json:"type"`
		} `json:"spec"`
	}
	if err := json.Unmarshal([]byte(src), &doc); err != nil {
		return "invalid JSON"
	}
	for _, kind := range []string{doc.ChartSpec.ChartType, doc.Type, doc.Spec.Type} {
		if kind != "" {
			return kind
		}
	}
	return "unknown chart"
}
