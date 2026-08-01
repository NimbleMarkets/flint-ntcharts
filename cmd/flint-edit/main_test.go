package main

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
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

	if strings.TrimSpace(m.chart) == "" {
		t.Fatal("no chart after the initial render")
	}
	if m.state != 1 {
		t.Fatalf("expected ok state after a valid spec, got %d (%q)", m.state, m.msg)
	}
	wide := m.chart

	next, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 16})
	m = renderNow(t, next.(model))
	if m.chart == wide {
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

	if len(examples) != 3 {
		t.Fatalf("expected 3 examples, got %d", len(examples))
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
		if strings.TrimSpace(m.chart) == "" {
			t.Fatalf("example %d (%s): empty chart", i+1, ex.name)
		}
	}
}
