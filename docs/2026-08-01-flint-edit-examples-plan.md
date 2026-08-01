# flint-edit Example Switcher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** flint-edit ships three built-in example specs (Bar, Timeseries Line, Candlestick), loadable with ctrl+1/2/3 or alt+1/2/3.

**Architecture:** A new `examples.go` in `cmd/flint-edit` holds an ordered `[]example` of compiled-in JSON specs; `Update`'s key handling intercepts the digit chords before the editor sees them, replaces the buffer via `ed.SetContent`, and fires the existing generation-guarded async render. No new dependencies, no new packages.

**Tech Stack:** Go 1.26.4, bubbletea v2.0.7 (`charm.land/bubbletea/v2`), goeditor v0.4.16, existing `tui.Render` seam.

**Spec:** `docs/2026-08-01-flint-edit-examples-design.md`

## Global Constraints

- Go directive stays `1.26.4`; **no new module dependencies**.
- Examples are string consts in `cmd/flint-edit/examples.go` — no `go:embed`, no runtime file reads, no imports from `testdata/`.
- Key strings are exactly what bubbletea v2 emits (verified against v2.0.7): `tea.KeyPressMsg{Code: '1', Mod: tea.ModAlt}.String() == "alt+1"`, and `"ctrl+2"` for `ModCtrl`. Bubble Tea v2 requests kitty key disambiguation from the terminal **by default**, so no `KeyboardEnhancements` opt-in is needed for ctrl+digit; do not add one.
- All three examples must compile through the real wasm compiler with **zero warnings** (status state 1, `compiled ok`) — the source fixtures (`bar-currency`, `line-temporal`, `candlestick`) are verified warning-free in `testdata/ntspec-golden-terminal/`.
- Every task ends with `go test ./cmd/flint-edit/ -count=1` green, `go vet ./...` clean, `gofmt -l cmd/flint-edit` empty.

---

### Task 1: `examples.go` — the example list

**Files:**
- Create: `cmd/flint-edit/examples.go`
- Modify: `cmd/flint-edit/main.go` (delete the `sample` const, seed from `examples[0]`)
- Test: `cmd/flint-edit/main_test.go`

**Interfaces:**
- Consumes: nothing new (uses the existing test harness `newTestModel` / `renderNow` in `main_test.go`).
- Produces: `type example struct { name, src string }` and `var examples []example` (len 3, index 0 = key 1), used by Task 2's key handling. `examples[0].src` is the startup buffer.

- [ ] **Step 1: Write the failing test**

Append to `cmd/flint-edit/main_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/flint-edit/ -run TestExamplesCompile -count=1`
Expected: FAIL to build with `undefined: examples`

- [ ] **Step 3: Create `cmd/flint-edit/examples.go`**

```go
package main

// example is a built-in demo spec, loadable with ctrl+N / alt+N where N is
// its 1-based position in examples.
type example struct {
	name string // shown on the status line
	src  string // full flint ChartAssemblyInput JSON
}

// examples is ordered: index 0 loads with key 1, and seeds the editor at
// startup. Sources are compiled in (adapted from testdata/fixtures-terminal,
// not embedded — the demo must not drift when test fixtures change) and must
// compile warning-free; TestExamplesCompile enforces that.
var examples = []example{
	{name: "Bar Chart", src: `{
  "data": { "values": [
    {"month": "Jan", "revenue": 120},
    {"month": "Feb", "revenue": 180},
    {"month": "Mar", "revenue": 95},
    {"month": "Apr", "revenue": 210},
    {"month": "May", "revenue": 165}
  ]},
  "semantic_types": { "revenue": {"semanticType": "Price", "unit": "USD"} },
  "chart_spec": {
    "chartType": "Bar Chart",
    "encodings": { "x": {"field": "month"}, "y": {"field": "revenue"} }
  }
}`},
	{name: "Timeseries Line", src: `{
  "data": { "values": [
    {"date": "2026-01-01", "price": 104.2},
    {"date": "2026-02-01", "price": 108.9},
    {"date": "2026-03-01", "price": 101.4},
    {"date": "2026-04-01", "price": 115.7},
    {"date": "2026-05-01", "price": 119.3},
    {"date": "2026-06-01", "price": 112.8},
    {"date": "2026-07-01", "price": 121.5}
  ]},
  "semantic_types": { "price": {"semanticType": "Price", "unit": "USD"} },
  "chart_spec": {
    "chartType": "Line Chart",
    "encodings": { "x": {"field": "date"}, "y": {"field": "price"} }
  }
}`},
	{name: "Candlestick", src: `{
  "data": { "values": [
    {"date": "2026-01-05", "open": 100, "high": 108, "low": 97,  "close": 105},
    {"date": "2026-01-06", "open": 105, "high": 112, "low": 103, "close": 110},
    {"date": "2026-01-07", "open": 110, "high": 111, "low": 98,  "close": 99},
    {"date": "2026-01-08", "open": 99,  "high": 106, "low": 96,  "close": 104},
    {"date": "2026-01-09", "open": 104, "high": 115, "low": 102, "close": 114},
    {"date": "2026-01-12", "open": 114, "high": 118, "low": 109, "close": 111},
    {"date": "2026-01-13", "open": 111, "high": 121, "low": 110, "close": 120}
  ]},
  "chart_spec": {
    "chartType": "Candlestick Chart",
    "encodings": {
      "x": {"field": "date"},
      "open": {"field": "open"}, "high": {"field": "high"},
      "low": {"field": "low"}, "close": {"field": "close"}
    }
  }
}`},
}
```

Note: the Timeseries Line data adds a `Price`/USD semantic type and a 7th
month over the fixture (demo polish); the Candlestick data extends the fixture
from 5 to 7 sessions. If either turns out to warn (Step 5), adjust the data —
the zero-warning bar is the requirement, the exact values are not.

- [ ] **Step 4: Absorb the old `sample` const in `main.go`**

Delete the `sample` const block (lines 28–41) and change the two uses in `newModel`:

```go
func newModel(c tui.Compiler) model {
	ed := goeditor.New(40, 20)
	ed.DisableVimMode(true) // plain, non-modal typing
	ed.SetLanguage("json", highlightTheme)
	ed.SetContent(examples[0].src)
	_ = ed.SetCursorPositionEnd()
	ed.Focus()
	return model{runner: c, ed: ed, src: examples[0].src, msg: "type to render"}
}
```

- [ ] **Step 5: Run the package tests**

Run: `go test ./cmd/flint-edit/ -count=1`
Expected: PASS (all tests, including the existing four). If `TestExamplesCompile` reports state 2 (warnings) for an example, fix that example's JSON until clean — do not weaken the test.

- [ ] **Step 6: Commit**

```bash
git add cmd/flint-edit/examples.go cmd/flint-edit/main.go cmd/flint-edit/main_test.go
git commit -m "feat(flint-edit): built-in example list; seed editor from examples[0]"
```

---

### Task 2: chord handling — ctrl/alt+digit loads an example

**Files:**
- Modify: `cmd/flint-edit/main.go` (the `tea.KeyMsg` case in `Update`; new `exampleChord` func and `loadExample` method)
- Test: `cmd/flint-edit/main_test.go`

**Interfaces:**
- Consumes: `examples`, `example` from Task 1; existing `renderCmd`, `renderedMsg`, generation counter `m.gen`.
- Produces: `exampleChord(key string) (int, bool)` (chord string → 0-based example index) and `(*model).loadExample(i int) tea.Cmd`. Task 3 relies on the switching behavior only, not these names.

- [ ] **Step 1: Write the failing test**

Append to `cmd/flint-edit/main_test.go`:

```go
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
	next, _ = m.Update(cmd())
	m = next.(model)
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/flint-edit/ -run TestExampleSwitching -count=1`
Expected: FAIL — `alt+2 did not switch src to example 2` (the chord currently falls through to the editor, which ignores it or types nothing; `src` stays example 1).

- [ ] **Step 3: Implement chord parsing and example loading in `main.go`**

Add below `renderCmd`:

```go
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
	m.ed.SetContent(ex.src)
	_ = m.ed.SetCursorPositionEnd()
	m.src = ex.src
	m.gen++
	m.msg, m.state = "loaded example: "+ex.name, 0
	return m.renderCmd()
}
```

Replace the `tea.KeyMsg` case in `Update`:

```go
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		if i, ok := exampleChord(key); ok && i < len(examples) {
			return m, m.loadExample(i)
		}
		// chords past the example list (e.g. alt+9) fall through to the editor
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/flint-edit/ -run TestExampleSwitching -count=1`
Expected: PASS

- [ ] **Step 5: Run the full package suite**

Run: `go test ./cmd/flint-edit/ -count=1 && go vet ./... && gofmt -l cmd/flint-edit`
Expected: tests PASS, vet clean, no gofmt output

- [ ] **Step 6: Commit**

```bash
git add cmd/flint-edit/main.go cmd/flint-edit/main_test.go
git commit -m "feat(flint-edit): ctrl/alt+1-3 load built-in examples"
```

---

### Task 3: title-bar hint + runbook

**Files:**
- Modify: `cmd/flint-edit/main.go` (`View`, title string)
- Modify: `docs/TESTING.md` (flint-edit manual-verify section)
- Test: `cmd/flint-edit/main_test.go` (extend `TestViewHasBothPanes`)

**Interfaces:**
- Consumes: switching behavior from Task 2.
- Produces: user-facing copy only.

- [ ] **Step 1: Extend the view test (failing first)**

In `TestViewHasBothPanes`, after the existing title-bar assertion, add:

```go
	if !strings.Contains(v.Content, "ctrl/alt+1·2·3 examples") {
		t.Fatal("title bar is missing the example-switcher hint")
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/flint-edit/ -run TestViewHasBothPanes -count=1`
Expected: FAIL with "title bar is missing the example-switcher hint"

- [ ] **Step 3: Update the title bar in `View`**

```go
	title := titleStyle.Render("flint-edit") +
		hintStyle.Render("   edit the spec, watch it render  ·  ctrl/alt+1·2·3 examples  ·  ctrl+c quit")
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/flint-edit/ -run TestViewHasBothPanes -count=1`
Expected: PASS

- [ ] **Step 5: Add the manual-verify step to `docs/TESTING.md`**

In the flint-edit section of `docs/TESTING.md`, append to the eyeball checklist:

```markdown
- Press `alt+2` (or `ctrl+2` in a kitty-protocol terminal): the buffer swaps
  to the Timeseries Line example and the chart re-renders; `alt+3` shows the
  Candlestick. `alt+1` returns to the Bar Chart. Unbound chords (`alt+9`) do
  nothing.
```

- [ ] **Step 6: Full-tree verification**

Run: `go test ./... -count=1 && go vet ./... && gofmt -l . | grep -v node_modules; true`
Expected: all packages PASS, vet clean, no gofmt output

- [ ] **Step 7: Commit**

```bash
git add cmd/flint-edit/main.go cmd/flint-edit/main_test.go docs/TESTING.md
git commit -m "feat(flint-edit): example-switcher hint in title bar; runbook step"
```
