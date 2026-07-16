package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
