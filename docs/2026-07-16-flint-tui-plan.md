# flint-tui Implementation Plan (flint → ntcharts, Phase 5)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `flint-tui` — a persistent terminal chart window: watches a file, reads a JSON stream from stdin, or listens on a Unix socket; agents push flint inputs (compiled via the embedded wasm), envelopes, or raw ntcharts-specs; the chart live-updates and re-fits on terminal resize; errors/warnings land in a status line, never kill the app.

**Architecture:** A `tui/` library package (testable without a TTY) + a thin `cmd/flint-tui/` main. All three sources are goroutines feeding `p.Send(InputMsg{...})`. Rendering is asynchronous (a `tea.Cmd` runs the 22 ms compile off the update loop). Resize strategy: **re-render from the retained raw document** — flint inputs re-compile with `WithBaseSize(w, h-1)`; envelope/direct specs get `Width/Height` overridden to fit the window, then `spec.Build` (Build returns pre-drawn models; no Resize/re-push dance needed). Plus one carried-over TS fix: palette stability across datasets (visible as color churn in a live TUI).

**Tech Stack:** `charm.land/bubbletea/v2 v2.0.7` (neomantra fork replace — ALREADY mirrored in go.mod), `charm.land/lipgloss/v2`, existing `compile` + ntcharts `spec` packages. No new JS deps; no fsnotify (mtime polling).

**Design refs:** design §5; ledger P5 notes (opaque error strings; palette instability).

## Global Constraints

- Work in `/Users/evan/projects/flint-ntcharts` (branch main). Explicit-path staging. Task 4 also touches nothing outside this repo.
- **bubbletea v2 API facts (verified against the resolved fork — do NOT drift toward v1 or charmbracelet v2-beta):** `Model` is `Init() Cmd; Update(Msg) (Model, Cmd); View() View`. `View()` returns the `tea.View` STRUCT — build via `tea.NewView(content string)`, then set `v.AltScreen = true` on it per frame (there is NO `WithAltScreen` program option). `tea.WindowSizeMsg{Width, Height int}`. Key handling: `case tea.KeyMsg:` then `switch msg.String() { case "q", "ctrl+c": return m, tea.Quit }`. External pushes: `p.Send(msg)` (safe after shutdown). Test options exist: `WithoutRenderer()`, `WithInput`, `WithOutput`, `WithWindowSize(w,h)`, `WithContext`.
- Import path is `charm.land/bubbletea/v2` (alias `tea`) — never `github.com/charmbracelet/bubbletea`.
- `spec.Build` returns pre-drawn models as `any` with a `View() string` method (same `viewer` idiom as speccheck). Do not call chart `Resize` — always re-Build/re-Compile from the retained document.
- Document sniffing rules (frozen): top-level `chart_spec` key → flint input (compile); top-level `spec` key → envelope (extract inner spec); top-level `type` key → ntcharts-spec (direct); anything else → status-line error. For envelope/direct docs, `Width`/`Height` are ALWAYS overridden to the chart area (fit-to-window; documented). For flint inputs the compiled spec's dimensions are respected (the backend already fits them to the requested base).
- Chart area = terminal width × (terminal height − 1); the bottom line is the status line. The status line shows: source tag, chart dimensions, warning count (+ first warning), or the current error (errors keep the last good chart on screen).
- Compile errors are opaque strings (P4 contract note) — display verbatim, never parse.
- The app must run with zero sources only if given a file arg, `--stdin`, or `--listen`; with none, print usage and exit 2 (not a TUI session).
- All suites green at each boundary: `go test ./...`, `go vet ./...`, and (Task 4 only) the JS chain + references + wasm consistency.
- Tests must not require a TTY (model-level Update/View tests + `WithoutRenderer` program smoke).

---

### Task 1: tui core — messages, sniffing, rendering, Model

**Files:**
- Create: `tui/tui.go`, `tui/render.go`
- Test: `tui/tui_test.go`, `tui/render_test.go`

**Interfaces:**
- Produces (Tasks 2-3 depend on exact names): `InputMsg{Raw []byte; Source string}`, `SourceErrMsg{Err error}`, `New(runner *compile.Runner) Model`, `Model` (implements `tea.Model`), and internal `renderDoc(runner, raw, w, h) renderedMsg`.

- [ ] **Step 1: Write the failing tests**

`tui/render_test.go`:

```go
package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
)

func newTestRunner(t *testing.T) *compile.Runner {
	t.Helper()
	r, err := compile.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })
	return r
}

const flintDoc = `{
  "data": {"values": [
    {"product": "A", "revenue": 10}, {"product": "B", "revenue": 25}, {"product": "C", "revenue": 15}
  ]},
  "chart_spec": {"chartType": "Bar Chart",
    "encodings": {"x": {"field": "product"}, "y": {"field": "revenue"}}}
}`

const specDoc = `{
  "type": "bar", "width": 30, "height": 8,
  "x_axis": {"labels": ["Q1", "Q2"]},
  "data": {"series": [{"name": "rev", "values": [{"y": 3}, {"y": 5}]}]}
}`

func TestSniff(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want docKind
	}{
		{"flint", flintDoc, docFlint},
		{"direct spec", specDoc, docSpec},
		{"envelope", `{"spec": {"type":"bar","width":10,"height":5,"data":{"series":[{"name":"a","values":[{"y":1}]}]}}, "warnings": [], "size": {"width":10,"height":5}}`, docEnvelope},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := sniff([]byte(c.doc))
			if err != nil || got != c.want {
				t.Fatalf("sniff = %v, %v; want %v", got, err, c.want)
			}
		})
	}
	if _, err := sniff([]byte(`{"foo": 1}`)); err == nil {
		t.Fatal("expected sniff error for unknown document shape")
	}
	if _, err := sniff([]byte(`not json`)); err == nil {
		t.Fatal("expected sniff error for invalid JSON")
	}
}

func TestRenderDocFlint(t *testing.T) {
	r := newTestRunner(t)
	msg := renderDoc(r, []byte(flintDoc), 60, 20)
	if msg.err != nil {
		t.Fatalf("renderDoc(flint): %v", msg.err)
	}
	if strings.TrimSpace(msg.view) == "" {
		t.Fatal("empty chart view")
	}
}

func TestRenderDocDirectSpecFitsWindow(t *testing.T) {
	r := newTestRunner(t)
	msg := renderDoc(r, []byte(specDoc), 44, 15)
	if msg.err != nil {
		t.Fatalf("renderDoc(spec): %v", msg.err)
	}
	lines := strings.Split(strings.TrimRight(msg.view, "\n"), "\n")
	if len(lines) > 14 { // fit: height-1 status line reserved by caller contract
		t.Fatalf("direct spec not fitted: %d lines > 14", len(lines))
	}
}

func TestRenderDocBadDocKeepsError(t *testing.T) {
	r := newTestRunner(t)
	msg := renderDoc(r, []byte(`{"chart_spec": {"chartType": "Rose Chart", "encodings": {}}, "data": {"values": []}}`), 40, 12)
	if msg.err == nil {
		t.Fatal("expected compile error to surface")
	}
	if !strings.Contains(msg.err.Error(), "Unknown chart type") {
		t.Fatalf("error should carry the compiler message verbatim, got: %v", msg.err)
	}
}
```

`tui/tui_test.go`:

```go
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
```

NOTE on `TestQuitKeys`: `tea.KeyPressMsg` is `type KeyPressMsg Key` — constructing one with `{Code: 'q'}` must satisfy `msg.String() == "q"`. Verify the `Key` struct's fields in the resolved fork (`~/go/pkg/mod/github.com/neomantra/bubbletea/v2@v2.0.0-20260506185856-6506c47fa2f3/key.go:302`) and construct accordingly (bounded adaptation; if constructing a KeyPressMsg is awkward, test the handler through a small exported-for-test helper instead — record the choice).

- [ ] **Step 2: Run tests to verify they fail** (`go test ./tui/` → package missing / undefined symbols)

- [ ] **Step 3: Implement `tui/render.go`**

```go
// Package tui implements the flint-tui live chart window: a bubbletea model
// that renders flint inputs (via the embedded wasm compiler), compile
// envelopes, or raw ntcharts-spec documents, re-fitting on terminal resize.
package tui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

type docKind int

const (
	docUnknown docKind = iota
	docFlint            // ChartAssemblyInput: has chart_spec
	docEnvelope         // compile envelope: has spec
	docSpec             // raw ntcharts-spec: has type
)

// sniff classifies a JSON document per the frozen sniffing rules.
func sniff(raw []byte) (docKind, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return docUnknown, fmt.Errorf("invalid JSON: %w", err)
	}
	switch {
	case probe["chart_spec"] != nil:
		return docFlint, nil
	case probe["spec"] != nil:
		return docEnvelope, nil
	case probe["type"] != nil:
		return docSpec, nil
	}
	return docUnknown, fmt.Errorf("document is neither flint input (chart_spec), envelope (spec), nor ntcharts-spec (type)")
}

// renderedMsg carries an async render result back into Update.
type renderedMsg struct {
	view     string
	warnings []compile.Warning
	err      error
}

type viewer interface{ View() string }

// renderDoc compiles/builds raw into a chart view sized w×h (the chart area,
// status line already excluded by the caller). Errors return err with view
// empty; the model decides what stays on screen.
func renderDoc(runner *compile.Runner, raw []byte, w, h int) renderedMsg {
	kind, err := sniff(raw)
	if err != nil {
		return renderedMsg{err: err}
	}
	var s spec.Spec
	var warnings []compile.Warning
	switch kind {
	case docFlint:
		s, warnings, err = runner.Compile(context.Background(), raw, compile.WithBaseSize(w, h))
		if err != nil {
			return renderedMsg{err: err}
		}
	case docEnvelope:
		var env struct {
			Spec spec.Spec `json:"spec"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			return renderedMsg{err: fmt.Errorf("bad envelope: %w", err)}
		}
		s = env.Spec
		s.Width, s.Height = w, h // fit-to-window
	case docSpec:
		if err := json.Unmarshal(raw, &s); err != nil {
			return renderedMsg{err: fmt.Errorf("bad ntcharts-spec: %w", err)}
		}
		s.Width, s.Height = w, h // fit-to-window
	}
	if err := s.Validate(); err != nil {
		return renderedMsg{err: err}
	}
	model, err := spec.Build(s)
	if err != nil {
		return renderedMsg{err: err}
	}
	v, ok := model.(viewer)
	if !ok {
		return renderedMsg{err: fmt.Errorf("chart model %T has no View", model)}
	}
	return renderedMsg{view: v.View(), warnings: warnings}
}
```

- [ ] **Step 4: Implement `tui/tui.go`**

```go
package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
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
}

func New(runner *compile.Runner) Model {
	return Model{runner: runner, waiting: true}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, m.rerenderCmd()
	case InputMsg:
		m.raw, m.source = msg.Raw, msg.Source
		m.waiting = false
		return m, m.rerenderCmd()
	case renderedMsg:
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
	runner, raw := m.runner, m.raw
	w, h := m.width, m.height-1 // reserve the status line
	return func() tea.Msg { return renderDoc(runner, raw, w, h) }
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
	switch {
	case m.err != nil:
		return left + "  " + errStyle.Render(truncate(m.err.Error(), m.width-lipgloss.Width(left)-2))
	case len(m.warnings) > 0:
		w := fmt.Sprintf("%d warning(s): %s", len(m.warnings), m.warnings[0].Message)
		return left + "  " + warnStyle.Render(truncate(w, m.width-lipgloss.Width(left)-2))
	}
	return left
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func truncate(s string, n int) string {
	if n <= 1 || len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
```

- [ ] **Step 5: Green + vet + commit**

```bash
cd /Users/evan/projects/flint-ntcharts
go test ./tui/ -v && go vet ./tui/ && go test ./...
git add tui/
git commit -m "feat(tui): core model - sniffing, async render, status line"
```

---

### Task 2: sources — file polling, stream decoding, unix socket

**Files:**
- Create: `tui/sources.go`
- Test: `tui/sources_test.go`

**Interfaces:**
- Produces: `WatchFile(ctx, path string, interval time.Duration, send func(tea.Msg))`, `ReadStream(ctx, r io.Reader, source string, send func(tea.Msg))`, `ListenSocket(ctx, path string, send func(tea.Msg)) error`. All send `InputMsg`/`SourceErrMsg`; all return promptly on ctx cancellation.

- [ ] **Step 1: Write the failing tests**

`tui/sources_test.go`:

```go
package tui

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func collector() (func(tea.Msg), chan tea.Msg) {
	ch := make(chan tea.Msg, 16)
	return func(m tea.Msg) { ch <- m }, ch
}

func waitInput(t *testing.T, ch chan tea.Msg) InputMsg {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case m := <-ch:
			if im, ok := m.(InputMsg); ok {
				return im
			}
		case <-deadline:
			t.Fatal("timed out waiting for InputMsg")
		}
	}
}

func TestReadStreamMultipleDocs(t *testing.T) {
	send, ch := collector()
	r := strings.NewReader(`{"type":"bar","width":10,"height":5,"data":{"series":[{"name":"a","values":[{"y":1}]}]}}
{"type":"bar","width":10,"height":5,"data":{"series":[{"name":"b","values":[{"y":2}]}]}}`)
	go ReadStream(context.Background(), r, "stdin", send)
	first := waitInput(t, ch)
	if !strings.Contains(string(first.Raw), `"a"`) {
		t.Fatalf("first doc wrong: %s", first.Raw)
	}
	second := waitInput(t, ch)
	if !strings.Contains(string(second.Raw), `"b"`) {
		t.Fatalf("second doc wrong: %s", second.Raw)
	}
}

func TestReadStreamPrettyPrintedDoc(t *testing.T) {
	send, ch := collector()
	r := strings.NewReader("{\n  \"type\": \"bar\",\n  \"width\": 10,\n  \"height\": 5,\n  \"data\": {\"series\": [{\"name\": \"a\", \"values\": [{\"y\": 1}]}]}\n}\n")
	go ReadStream(context.Background(), r, "stdin", send)
	got := waitInput(t, ch)
	if !strings.Contains(string(got.Raw), `"bar"`) {
		t.Fatalf("pretty doc not decoded: %s", got.Raw)
	}
}

func TestWatchFileDetectsChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chart.json")
	if err := os.WriteFile(path, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	send, ch := collector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go WatchFile(ctx, path, 20*time.Millisecond, send)
	first := waitInput(t, ch)
	if !strings.Contains(string(first.Raw), `"v":1`) {
		t.Fatalf("initial read wrong: %s", first.Raw)
	}
	// mtime granularity: rewrite with different content and a nudge
	time.Sleep(30 * time.Millisecond)
	if err := os.WriteFile(path, []byte(`{"v":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	second := waitInput(t, ch)
	if !strings.Contains(string(second.Raw), `"v":2`) {
		t.Fatalf("change not detected: %s", second.Raw)
	}
}

func TestListenSocketDeliversDocs(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "flint.sock")
	send, ch := collector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := ListenSocket(ctx, sock, send); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write([]byte(`{"type":"bar","width":10,"height":5,"data":{"series":[{"name":"s","values":[{"y":9}]}]}}` + "\n")); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	got := waitInput(t, ch)
	if got.Source == "" || !strings.Contains(string(got.Raw), `"s"`) {
		t.Fatalf("socket doc wrong: source=%q raw=%s", got.Source, got.Raw)
	}
}
```

- [ ] **Step 2: RED** (`go test ./tui/ -run 'TestReadStream|TestWatchFile|TestListenSocket'` → undefined)

- [ ] **Step 3: Implement `tui/sources.go`**

```go
package tui

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
)

// ReadStream decodes a sequence of JSON documents (NDJSON or pretty-printed,
// json.Decoder handles both) and sends each as an InputMsg. Returns on EOF,
// decode error, or ctx cancellation (via the reader being closed by its owner).
func ReadStream(ctx context.Context, r io.Reader, source string, send func(tea.Msg)) {
	dec := json.NewDecoder(r)
	for {
		var raw json.RawMessage
		err := dec.Decode(&raw)
		if err != nil {
			if !errors.Is(err, io.EOF) && ctx.Err() == nil {
				send(SourceErrMsg{Err: err})
			}
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		send(InputMsg{Raw: raw, Source: source})
	}
}

// WatchFile polls path every interval and sends the file's content whenever
// its mtime or size changes (including the initial read). Missing files are
// tolerated (the file may appear later or be mid-rewrite).
func WatchFile(ctx context.Context, path string, interval time.Duration, send func(tea.Msg)) {
	var lastMod time.Time
	var lastSize int64
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		if st, err := os.Stat(path); err == nil {
			if !st.ModTime().Equal(lastMod) || st.Size() != lastSize {
				lastMod, lastSize = st.ModTime(), st.Size()
				if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
					send(InputMsg{Raw: b, Source: "file:" + path})
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// ListenSocket creates a unix socket at path (removing any stale one) and
// serves each connection as a document stream. Returns an error only if the
// initial listen fails; per-connection errors go to send as SourceErrMsg.
func ListenSocket(ctx context.Context, path string, send func(tea.Msg)) error {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				if ctx.Err() == nil {
					send(SourceErrMsg{Err: err})
				}
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				ReadStream(ctx, c, "socket:"+path, send)
			}(conn)
		}
	}()
	return nil
}
```

- [ ] **Step 4: Green + full suite + commit**

```bash
cd /Users/evan/projects/flint-ntcharts
go test ./tui/ -v && go vet ./tui/ && go test ./...
git add tui/sources.go tui/sources_test.go
git commit -m "feat(tui): file/stdin/socket sources"
```

---

### Task 3: cmd/flint-tui + program smoke + CI/docs

**Files:**
- Create: `cmd/flint-tui/main.go`
- Test: `cmd/flint-tui/main_test.go` (flag/config parsing) + `tui/program_test.go` (headless program smoke)
- Modify: `.github/workflows/ci.yml` (add `go build ./...` to the guarded Go step), `README.md` (flint-tui usage section)

**Interfaces:**
- Produces: the `flint-tui` binary. CLI: `flint-tui [flags] [file]` — positional file to watch; `--stdin` read document stream from stdin; `--listen PATH` unix socket; `--poll DUR` (default 250ms) file poll interval. At least one source required, else usage + exit 2. Multiple sources may be combined (latest document wins).

- [ ] **Step 1: Write the failing tests**

`cmd/flint-tui/main_test.go`:

```go
package main

import "testing"

func TestParseConfig(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(cfg config) bool
	}{
		{"file arg", []string{"chart.json"}, false, func(c config) bool { return c.file == "chart.json" }},
		{"stdin", []string{"--stdin"}, false, func(c config) bool { return c.stdin }},
		{"socket", []string{"--listen", "/tmp/f.sock"}, false, func(c config) bool { return c.socket == "/tmp/f.sock" }},
		{"combined", []string{"--stdin", "chart.json"}, false, func(c config) bool { return c.stdin && c.file == "chart.json" }},
		{"no sources", []string{}, true, nil},
		{"two positionals", []string{"a.json", "b.json"}, true, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := parseConfig(c.args)
			if c.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if err == nil && !c.check(cfg) {
				t.Fatalf("config check failed: %+v", cfg)
			}
		})
	}
}
```

`tui/program_test.go` (headless end-to-end: real tea.Program, no TTY):

```go
package tui

import (
	"bytes"
	"context"
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
	_ = context.Background()
}
```

(Adapt the smoke to the fork's actual behavior if `WithWindowSize` doesn't emit a WindowSizeMsg under WithoutRenderer — fall back to `p.Send(tea.WindowSizeMsg{Width: 60, Height: 20})` before the InputMsg; record the choice.)

- [ ] **Step 2: RED**, then implement `cmd/flint-tui/main.go`

```go
// Command flint-tui is a persistent terminal chart window. Agents and scripts
// push flint inputs, compile envelopes, or ntcharts-spec documents via a
// watched file, stdin, or a unix socket; the newest document is rendered and
// re-fitted on every terminal resize.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/tui"
)

type config struct {
	file   string
	stdin  bool
	socket string
	poll   time.Duration
}

func parseConfig(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("flint-tui", flag.ContinueOnError)
	fs.BoolVar(&cfg.stdin, "stdin", false, "read a JSON document stream from stdin")
	fs.StringVar(&cfg.socket, "listen", "", "listen on a unix socket `path` for document streams")
	fs.DurationVar(&cfg.poll, "poll", 250*time.Millisecond, "file poll interval")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	switch fs.NArg() {
	case 0:
	case 1:
		cfg.file = fs.Arg(0)
	default:
		return cfg, fmt.Errorf("at most one file argument, got %d", fs.NArg())
	}
	if cfg.file == "" && !cfg.stdin && cfg.socket == "" {
		return cfg, fmt.Errorf("no source: pass a file, --stdin, or --listen")
	}
	return cfg, nil
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "flint-tui: %v\nusage: flint-tui [--stdin] [--listen SOCK] [--poll 250ms] [chart.json]\n", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner, err := compile.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "flint-tui: compiler init: %v\n", err)
		os.Exit(1)
	}
	defer runner.Close(ctx)

	opts := []tea.ProgramOption{tea.WithContext(ctx)}
	if cfg.stdin {
		// stdin carries data, not key input; use the TTY for keys if available.
		tty, err := os.Open("/dev/tty")
		if err == nil {
			defer tty.Close()
			opts = append(opts, tea.WithInput(tty))
		}
	}
	p := tea.NewProgram(tui.New(runner), opts...)

	send := func(msg tea.Msg) { p.Send(msg) }
	if cfg.file != "" {
		go tui.WatchFile(ctx, cfg.file, cfg.poll, send)
	}
	if cfg.stdin {
		go tui.ReadStream(ctx, os.Stdin, "stdin", send)
	}
	if cfg.socket != "" {
		if err := tui.ListenSocket(ctx, cfg.socket, send); err != nil {
			fmt.Fprintf(os.Stderr, "flint-tui: listen: %v\n", err)
			os.Exit(1)
		}
		defer os.Remove(cfg.socket)
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "flint-tui: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 3: CI + README**

- ci.yml guarded Go step: add `go build ./...` before `go test ./...` (the cmd package only compiles with the sibling present — it imports ntcharts spec transitively; it's inside the guard already).
- README: "flint-tui" section — usage lines for all three sources, the sniffing rules table, fit-to-window semantics, an agent-workflow example (`flint-chart-mcp compile → jq → socket` or simply `cat chart.json > watched.json`), keys (q to quit), and a note that the status line shows compiler errors verbatim while keeping the last good chart.

- [ ] **Step 4: Green + manual smoke + commit**

```bash
cd /Users/evan/projects/flint-ntcharts
go test ./... && go vet ./... && go build ./cmd/flint-tui
printf '%s' '{"data":{"values":[{"p":"A","r":3},{"p":"B","r":7}]},"chart_spec":{"chartType":"Bar Chart","encodings":{"x":{"field":"p"},"y":{"field":"r"}}}}' > /tmp/flint-tui-demo.json
# Manual check is NOT possible in this harness (no TTY) — verify the binary
# starts and exits cleanly instead:
./flint-tui /tmp/flint-tui-demo.json < /dev/null > /dev/null 2>&1 || true  # must not panic; record observed behavior
git add cmd/ tui/program_test.go .github/workflows/ci.yml README.md
git commit -m "feat(cmd): flint-tui binary - sources wiring, flags, docs"
```

Note for the human: a real interactive check (`./flint-tui /tmp/flint-tui-demo.json`, then edit the file and watch it update; resize the window) is worth doing once — record in the report that it awaits a human TTY session.

---

### Task 4: palette stability across datasets (P3 carry-over)

**Files:**
- Modify: `js/src/ntcharts/series.ts`
- Modify: `js/test/golden.test.ts` (regression test)
- Possibly regenerated: goldens/references + `compile/flint.wasm` + buildinfo (bundle changes → rebuild)

**Interfaces:**
- Series colors become stable per category-order position: a category keeps its palette color even when other categories drop out of a later dataset (live TUI updates no longer churn colors).

- [ ] **Step 1: Failing test**

Append to `js/test/golden.test.ts`:

```ts
describe("series palette stability", () => {
  const mk = (rows: Array<Record<string, unknown>>) => assembleNtcharts({
    data: { values: rows },
    chart_spec: {
      chartType: "Line Chart",
      encodings: { x: { field: "t" }, y: { field: "v" }, color: { field: "cat" } },
      baseSize: { width: 40, height: 12 },
    },
  } as any);
  it("keeps a category's color when another category disappears", () => {
    const full = mk([
      { t: 1, v: 1, cat: "alpha" }, { t: 1, v: 2, cat: "beta" }, { t: 1, v: 3, cat: "gamma" },
      { t: 2, v: 2, cat: "alpha" }, { t: 2, v: 3, cat: "beta" }, { t: 2, v: 4, cat: "gamma" },
    ]);
    const partial = mk([
      { t: 1, v: 1, cat: "alpha" }, { t: 1, v: 3, cat: "gamma" },
      { t: 2, v: 2, cat: "alpha" }, { t: 2, v: 4, cat: "gamma" },
    ]);
    const colorOf = (out: any, name: string) =>
      out.data.series.find((s: any) => s.name === name)?.color;
    expect(colorOf(partial, "alpha")).toBe(colorOf(full, "alpha"));
    expect(colorOf(partial, "gamma")).toBe(colorOf(full, "gamma"));
  });
});
```

CAVEAT: with no `ordinalSortOrder`, order comes from first appearance — in `partial`, gamma is the SECOND series, so today it gets beta's old color (index 1)… which equals the full dataset's index-1 color — i.e. this test as written FAILS today for `gamma` (it gets palette[1] instead of palette[2]) only if first-appearance order differs. Verify RED actually reproduces; if flint's pivot/ordinalSortOrder resolution gives a stable canonical order that already makes this pass, find the real churn case (e.g. reversed appearance order) and adjust the fixture — the test must be RED before the fix. Record what you found.

- [ ] **Step 2: Fix `splitSeries`**

In `js/src/ntcharts/series.ts`: colors must be assigned from each name's position in the full `order` list BEFORE empty-bucket filtering, and — when no explicit `ordinalSortOrder` exists — order alone can't be stable across datasets, so ALSO make color assignment deterministic by name when order isn't authoritative. Implement: `hashColor(name)` fallback? NO — keep it simple and predictable: color index = position in `ordinalSortOrder` when flint resolved one, else position in first-appearance order of THIS dataset (unchanged), BUT the filter no longer shifts indices:

```ts
return order
  .map((name, i) => ({ name, values: buckets.get(name) ?? [], color: palette[i % palette.length] }))
  .filter((s) => s.values.length > 0);
```

(Assign color before filtering — an empty preferred bucket no longer shifts later series' colors. Cross-dataset stability is thus guaranteed whenever flint resolves an ordinalSortOrder — the common case for repeated datasets of the same shape; document the remaining first-appearance caveat in a comment.)

- [ ] **Step 3: GREEN + regenerate everything consistently**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npx vitest run              # palette test green; check whether any golden changed
UPDATE_GOLDEN=1 npx vitest run   # only if goldens legitimately changed — inspect diffs (colors only)
npm run gen-expected        # references
cd .. && make wasm          # bundle changed → rebuild + buildinfo
go test ./...               # parity + speccheck + tui green against new wasm
```

If goldens/references changed: color values only — anything else, STOP.

- [ ] **Step 4: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add js/src/ntcharts/series.ts js/test/golden.test.ts compile/flint.wasm compile/flint.wasm.buildinfo
# plus testdata/ paths ONLY if regenerated
git commit -m "fix(ts): stable series palette across datasets; rebuild wasm"
```
