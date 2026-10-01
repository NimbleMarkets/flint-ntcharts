package tui

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func drainCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		var next tea.Model
		next, cmd = m.Update(msg)
		m = next.(Model)
	}
	return m
}

func TestModelLifecycle(t *testing.T) {
	r := newTestRunner(t)
	m := New(r)

	// window size first (bubbletea always sends it)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = next.(Model)

	// input arrives → async render cmd → renderedMsg
	next, cmd := m.Update(InputMsg{Raw: []byte(flintDoc), Source: "test"})
	m = drainCmd(t, next.(Model), cmd)

	v := m.View()
	if !v.AltScreen {
		t.Fatal("View must set AltScreen")
	}
	if strings.TrimSpace(v.Content) == "" {
		t.Fatal("View content empty after successful render")
	}
	if !strings.Contains(v.Content, "test") {
		t.Fatal("status line should show the source tag")
	}
}

func TestModelErrorKeepsLastGoodChart(t *testing.T) {
	r := newTestRunner(t)
	m := New(r)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = next.(Model)
	next, cmd := m.Update(InputMsg{Raw: []byte(flintDoc), Source: "test"})
	m = drainCmd(t, next.(Model), cmd)
	goodView := m.View().Content

	next, cmd = m.Update(InputMsg{Raw: []byte(`{"nope": true}`), Source: "test"})
	m = drainCmd(t, next.(Model), cmd)
	v := m.View().Content
	if !strings.Contains(v, "neither flint input") {
		t.Fatalf("status line should show the sniff error, got:\n%s", v)
	}
	// the chart body (everything above the status line) must be unchanged
	chartOf := func(s string) string {
		lines := strings.Split(s, "\n")
		return strings.Join(lines[:len(lines)-1], "\n")
	}
	if chartOf(v) != chartOf(goodView) {
		t.Fatal("error must keep the last good chart on screen")
	}
}

func TestModelResizeRerenders(t *testing.T) {
	r := newTestRunner(t)
	m := New(r)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = next.(Model)
	next, cmd := m.Update(InputMsg{Raw: []byte(flintDoc), Source: "test"})
	m = drainCmd(t, next.(Model), cmd)
	wide := m.View().Content

	next, cmd = m.Update(tea.WindowSizeMsg{Width: 34, Height: 12})
	m = drainCmd(t, next.(Model), cmd)
	narrow := m.View().Content
	if wide == narrow {
		t.Fatal("resize must re-render the chart at the new size")
	}
}

func TestQuitKeys(t *testing.T) {
	r := newTestRunner(t)
	m := New(r)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("q must quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("q must produce tea.Quit")
	}
}

// TestModelDropsStaleRenderedMsg simulates the out-of-order-completion race:
// two documents are pushed back-to-back (A then B), and their renders
// complete out of order (B's result arrives before A's, since renders run
// concurrently on separate goroutines and compile time varies). The model
// must keep B's render — the latest request — and discard the stale A
// result even though it is delivered last.
func TestModelDropsStaleRenderedMsg(t *testing.T) {
	r := newTestRunner(t)
	m := New(r)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = next.(Model)

	next, cmdA := m.Update(InputMsg{Raw: []byte(flintDoc), Source: "A"})
	m = next.(Model)
	if cmdA == nil {
		t.Fatal("expected a render cmd for input A")
	}

	next, cmdB := m.Update(InputMsg{Raw: []byte(specDoc), Source: "B"})
	m = next.(Model)
	if cmdB == nil {
		t.Fatal("expected a render cmd for input B")
	}

	// Invoke the captured cmds directly to build the renderedMsgs, then
	// deliver B's result (the later request) before A's (the earlier,
	// now-superseded request) — the out-of-order arrival the guard defends
	// against.
	msgB := cmdB()
	next, _ = m.Update(msgB)
	m = next.(Model)
	viewAfterB := m.pane.View()
	if strings.TrimSpace(viewAfterB) == "" {
		t.Fatal("expected B's render to be applied")
	}

	msgA := cmdA()
	next, _ = m.Update(msgA)
	m = next.(Model)

	if m.pane.View() != viewAfterB {
		t.Fatal("stale render for input A must not overwrite the newer B render")
	}
	if m.err != nil {
		t.Fatalf("unexpected error after dropping stale render: %v", m.err)
	}
}

func TestStatusLineUnicodeSafe(t *testing.T) {
	r := newTestRunner(t)
	m := New(r)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 8})
	m = next.(Model)
	m.err = fmt.Errorf("ошибка: поле «цена» не число — очень длинное сообщение для усечения")
	line := m.statusLine()
	if !utf8.ValidString(line) {
		t.Fatal("status line must remain valid UTF-8 after truncation")
	}
}

// TestStatusLineWholeLineClamped covers the case where "left" alone (source
// tag + dimensions) already overflows a narrow terminal width, e.g. a long
// file path — the whole assembled line must still be clamped to m.width as a
// safety net on top of the existing suffix budgeting, or the status line
// wraps onto a second row.
func TestStatusLineWholeLineClamped(t *testing.T) {
	r := newTestRunner(t)
	m := New(r)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	m = next.(Model)
	next, _ = m.Update(InputMsg{
		Raw:    []byte(flintDoc),
		Source: "/very/long/path/to/some/deeply/nested/chart-document.json",
	})
	m = next.(Model)
	line := m.statusLine()
	if w := lipgloss.Width(line); w > 20 {
		t.Fatalf("status line width = %d, want <= 20; line: %q", w, line)
	}
}
