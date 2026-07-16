package tui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestProgramHeadlessSmoke(t *testing.T) {
	r := newTestRunner(t)
	m := New(r)
	var out bytes.Buffer
	p := tea.NewProgram(m,
		tea.WithoutRenderer(),
		tea.WithoutSignals(),
		tea.WithInput(strings.NewReader("")),
		tea.WithOutput(&out),
		tea.WithWindowSize(60, 20),
	)
	done := make(chan struct{})
	var final tea.Model
	go func() {
		defer close(done)
		final, _ = p.Run()
	}()
	p.Send(InputMsg{Raw: []byte(flintDoc), Source: "smoke"})
	time.Sleep(300 * time.Millisecond) // allow async render
	p.Quit()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("program did not quit")
	}
	fm, ok := final.(Model)
	if !ok {
		t.Fatalf("final model %T", final)
	}
	if strings.TrimSpace(fm.chartView) == "" {
		t.Fatal("program never rendered the pushed document")
	}
}
