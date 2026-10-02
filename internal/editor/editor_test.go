package editor

import (
	"context"
	"image"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/tui"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

func newTestModel(t *testing.T) model {
	t.Helper()
	r, err := compile.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })
	return newModel(r)
}

// renderNow runs the model's own render command synchronously and feeds the
// result back. It deliberately does NOT drive the editor's commands (cursor
// blink etc.), which the real bubbletea program runs on its own goroutines.
func renderNow(t *testing.T, m model) model {
	t.Helper()
	cmd := m.renderCmd()
	if cmd == nil {
		t.Fatal("no render command (window not sized?)")
	}
	next, _ := m.Update(cmd())
	return next.(model)
}

func TestEditorRendersOnResize(t *testing.T) {
	m := newTestModel(t)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = renderNow(t, next.(model))

	if strings.TrimSpace(m.pane.View()) == "" {
		t.Fatal("no chart after the initial render")
	}
	if m.state != 1 {
		t.Fatalf("expected ok state after a valid spec, got %d (%q)", m.state, m.msg)
	}
	wide := m.pane.View()

	next, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 16})
	m = renderNow(t, next.(model))
	if m.pane.View() == wide {
		t.Fatal("resize did not re-render the chart at the new size")
	}
}

func TestViewHasBothPanes(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = renderNow(t, next.(model))

	v := m.View()
	if !v.AltScreen {
		t.Fatal("View must request the alt screen")
	}
	if !strings.Contains(v.Content, "flint-edit") {
		t.Fatal("View is missing the title bar")
	}
	if !strings.Contains(v.Content, "compiled ok") {
		t.Fatal("View is missing the status line")
	}
	if !strings.Contains(v.Content, "│") {
		t.Fatal("View is missing the pane divider")
	}
	if !strings.Contains(v.Content, "ctrl+n next example") {
		t.Fatal("title bar is missing the example-switcher hint")
	}
}

// TestEditorHighlights confirms the JSON syntax highlighting is wired: the
// window-size tick is forwarded into the editor's Update, which lays out and
// paints colored tokens (goeditor emits truecolor even to a pipe, so this
// holds without a TTY).
func TestEditorHighlights(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = next.(model)

	if !strings.Contains(m.ed.View(), "\x1b[38") {
		t.Fatal("editor pane has no syntax-highlight color — is WindowSizeMsg forwarded to the editor?")
	}
}

// TestExamplesCompile guards against a typo'd built-in example shipping
// broken: every example must compile through the real wasm compiler with a
// clean status (no warnings) and render a non-empty chart.
func TestExamplesCompile(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = next.(model)

	if len(examples) != 8 {
		t.Fatalf("expected 8 examples, got %d", len(examples))
	}
	for i, want := range []string{"Bar Chart", "Timeseries Line", "Candlestick", "Log Scale", "Heatmap", "Histogram", "Calendar Heatmap", "Grouped Bars (raster)"} {
		if examples[i].name != want {
			t.Fatalf("examples[%d].name = %q, want %q", i, examples[i].name, want)
		}
	}
	if m.src != examples[0].src {
		t.Fatal("startup buffer is not examples[0]")
	}
	for i, ex := range examples {
		m.src = ex.src
		m.gen++
		m = renderNow(t, m)
		if m.state != 1 {
			t.Fatalf("example %d (%s): state %d, msg %q — want clean compile",
				i+1, ex.name, m.state, m.msg)
		}
		if strings.TrimSpace(m.pane.View()) == "" {
			t.Fatalf("example %d (%s): empty chart", i+1, ex.name)
		}
	}
}

// TestExampleSwitching drives ctrl/alt+digit chords through Update and
// asserts the buffer, render pipeline, and out-of-range behavior.
func TestExampleSwitching(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m = next.(model)

	// alt+2 → example 2 in buffer and src, render fired and lands clean
	next, cmd := m.Update(tea.KeyPressMsg{Code: '2', Mod: tea.ModAlt})
	m = next.(model)
	if m.src != examples[1].src {
		t.Fatal("alt+2 did not switch src to example 2")
	}
	if m.ed.GetCurrentContent() != examples[1].src {
		t.Fatal("alt+2 did not replace the editor buffer")
	}
	if !strings.Contains(m.msg, examples[1].name) {
		t.Fatalf("status %q does not name the loaded example", m.msg)
	}
	if cmd == nil {
		t.Fatal("alt+2 did not fire a render")
	}
	// The returned command batches the render with the editor's own repaint
	// commands; run just the render (see renderNow) rather than the batch.
	m = renderNow(t, m)
	if m.state != 1 {
		t.Fatalf("example 2 render not clean: state %d, msg %q", m.state, m.msg)
	}

	// ctrl+3 → example 3 (same path, ctrl modifier)
	next, cmd = m.Update(tea.KeyPressMsg{Code: '3', Mod: tea.ModCtrl})
	m = next.(model)
	if m.src != examples[2].src || cmd == nil {
		t.Fatal("ctrl+3 did not switch to example 3 with a render")
	}

	// alt+9 → no example there; buffer untouched
	before := m.src
	next, _ = m.Update(tea.KeyPressMsg{Code: '9', Mod: tea.ModAlt})
	m = next.(model)
	if m.src != before {
		t.Fatal("alt+9 must not change the buffer")
	}
}

// TestExampleSwitchRepaintsEditor is the on-screen half of example switching:
// the buffer changing is not enough, the editor pane the user is looking at
// must show the new example straight away, without waiting for a keystroke.
func TestExampleSwitchRepaintsEditor(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(model)
	// Highlighting splits tokens with colour codes; compare the plain text.
	if !strings.Contains(ansi.Strip(m.ed.View()), "revenue") {
		t.Fatal("precondition: editor pane should show example 1 (field \"revenue\")")
	}

	next, _ = m.Update(tea.KeyPressMsg{Code: '3', Mod: tea.ModAlt})
	m = next.(model)
	pane := ansi.Strip(m.ed.View())
	if !strings.Contains(pane, "Candlestick") {
		t.Fatal("after alt+3 the editor pane does not show example 3 (\"Candlestick Chart\")")
	}
	if strings.Contains(pane, "revenue") {
		t.Fatal("after alt+3 the editor pane still shows example 1 (field \"revenue\")")
	}
}

// TestNextExampleCycles covers ctrl+n, the chord every terminal delivers
// (ctrl+digit needs the kitty keyboard protocol, alt+digit needs
// option-as-meta on macOS): it steps through the examples and wraps.
func TestNextExampleCycles(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(model)

	ctrlN := tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl}
	for step, want := range []int{1, 2, 3, 4, 5, 6, 7, 0, 1} {
		next, cmd := m.Update(ctrlN)
		m = next.(model)
		if m.src != examples[want].src {
			t.Fatalf("ctrl+n step %d: expected example %d (%s)", step+1, want+1, examples[want].name)
		}
		if cmd == nil {
			t.Fatalf("ctrl+n step %d did not fire a render", step+1)
		}
		if !strings.Contains(m.msg, examples[want].name) {
			t.Fatalf("ctrl+n step %d: status %q does not name the example", step+1, m.msg)
		}
	}

	// A direct jump resets where ctrl+n continues from: after jumping to the
	// last example, ctrl+n wraps to the first.
	last := len(examples)
	next, _ = m.Update(tea.KeyPressMsg{Code: rune('0' + last), Mod: tea.ModAlt})
	m = next.(model)
	if m.src != examples[last-1].src {
		t.Fatalf("alt+%d did not load the last example", last)
	}
	next, _ = m.Update(ctrlN)
	m = next.(model)
	if m.src != examples[0].src {
		t.Fatal("ctrl+n after the last example should wrap to example 1")
	}
}

func TestExampleChord(t *testing.T) {
	cases := []struct {
		key string
		idx int
		ok  bool
	}{
		{"ctrl+1", 0, true},
		{"alt+1", 0, true},
		{"alt+9", 8, true},
		{"ctrl+c", 0, false},
		{"alt+", 0, false},
		{"ctrl+0", 0, false},
		{"ctrl+alt+1", 0, false},
		{"1", 0, false},
	}
	for _, c := range cases {
		idx, ok := exampleChord(c.key)
		if ok != c.ok || (ok && idx != c.idx) {
			t.Errorf("exampleChord(%q) = (%d, %v), want (%d, %v)", c.key, idx, ok, c.idx, c.ok)
		}
	}
}

// ctrl+g must never look like a dead key: it reports the mode it chose, or why
// it could not change it.
func TestCtrlGReportsOnTheStatusLine(t *testing.T) {
	prev := picture.KittySupported()
	t.Cleanup(func() { picture.ForceKittyCapability(prev) })

	m := newTestModel(t)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = next.(model)
	m.pane.Apply(tui.Frame{Image: image.NewRGBA(image.Rect(0, 0, 160, 96))})
	ctrlG := tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}

	picture.ForceKittyCapability(picture.KittyCapabilityUnsupported)
	next, _ = m.Update(ctrlG)
	m = next.(model)
	if !strings.Contains(m.msg, "FLINT_KITTY=1") {
		t.Fatalf("status %q should explain why Kitty is unavailable", m.msg)
	}

	picture.ForceKittyCapability(picture.KittyCapabilitySupported)
	next, _ = m.Update(ctrlG)
	m = next.(model)
	if !strings.Contains(m.msg, "Kitty graphics") {
		t.Fatalf("status %q should name the new mode", m.msg)
	}
}
