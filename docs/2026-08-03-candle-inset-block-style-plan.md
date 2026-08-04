# Candle Edge Insets & Block Style Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Edge candles render whole bodies (inset, not clipped) and specs can select solid-block candle bodies via `options.candle_style`, reachable from flint chart specs.

**Architecture:** Task 1 adds the block primitive (`DrawCandlestickBlockBottomToTop`) + the `▀` rune. Task 2 refactors `DrawCandleWidth` into `DrawCandleWithOpts{Width, Block}` with center-clamping edge insets. Task 3 adds `Options.CandleStyle` to ntcharts-spec and threads it through `buildOHLC`. Task 4 is the flint side: template passthrough of `chartProperties.candleStyle`, demo example update, wasm rebuild with goldens asserted byte-identical.

**Tech Stack:** Go (ntcharts), TypeScript (flint-ntcharts backend), esbuild/Javy wasm train.

**Spec:** `docs/2026-08-03-candle-inset-block-style-design.md`

## Global Constraints

- Tasks 1–3 change only `/Users/evan/projects/ntcharts` (branch `spec`, checked out — never switch/pull/push). Task 4 changes only flint-ntcharts, on branch `candle-block-style` (created during setup).
- Never stage ntcharts' untracked `examples/combo_chart.html`, nor flint-ntcharts' user-owned uncommitted files (`LICENSE.txt`, `README.md`, `js/package.json`, `evan.md`). Explicit paths only; no `git add -A`/`.`/`-a`.
- Block↔line rune mapping is mechanical parity with the line style's half-rounding: `┃→█ (FullBlock)`, `╻→▄ (LowerBlockFour)`, `╹→▀ (UpperHalfBlock, new const '▀')`; wick cells stay light line runes (`│ ╵ ╷`); at shared body/wick cells the block wins.
- Inset formula: `half=(w−1)/2`, `right=w−1−half`, clamp center into `[minCol+half, Canvas.Width()−1−right]`, skipped when that interval is empty; width 1 must be behavior-identical to today.
- `Options.CandleStyle`: `""`/`"line"` current behavior, `"block"` block bodies, anything else → `Build` error `spec: unknown candle_style %q`.
- Task 4: all goldens (`testdata/ntspec-golden{,-terminal}/`) and Node reference fixtures (`testdata/expected{,-terminal}/`) must remain byte-identical (`git status --short testdata/` empty after regen runs) — the candlestick fixture does not set the property.
- Each ntcharts task ends green: `go test ./canvas/... ./linechart/... ./spec/ -count=1`, `go vet ./...` clean, `gofmt -l canvas linechart spec` empty. Task 4 ends with vitest + tsc green, `make wasm` provenance rebuilt, and full Go suites green in BOTH repos.

---

### Task 1: block primitive (`canvas/graph` + `canvas/runes`)

**Files:**
- Modify: `/Users/evan/projects/ntcharts/canvas/runes/runes.go` (add `UpperHalfBlock` beside the block constants at lines 38-45)
- Modify: `/Users/evan/projects/ntcharts/canvas/graph/graph.go` (add function after `DrawCandlestickBottomToTopWide`, ~line 914)
- Test: `/Users/evan/projects/ntcharts/canvas/graph/candlestick_block_test.go` (create)

**Interfaces:**
- Consumes: `canvas.Model.SetCell`, `canvas.NewCellWithStyle`, existing rune constants.
- Produces: `DrawCandlestickBlockBottomToTop(m *canvas.Model, p canvas.Point, l, bl, bh, h float64, s lipgloss.Style)` and `runes.UpperHalfBlock` — Task 2 calls the primitive.

- [ ] **Step 1: Write the failing test**

Create `/Users/evan/projects/ntcharts/canvas/graph/candlestick_block_test.go`:

```go
// ntcharts - Copyright (c) 2026 Neomantra Corp.

package graph

import (
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"

	"charm.land/lipgloss/v2"
)

// Block style: body interior █, boundary cells ▄/▀ by the line style's
// half-rounding, wick cells light line runes.
func TestDrawCandlestickBlockRunes(t *testing.T) {
	s := lipgloss.NewStyle()
	c := canvas.New(3, 6)
	// l=0.0 → wick end row 0-from-bottom; body bl=1.0..bh=4.0; h=5.0
	DrawCandlestickBlockBottomToTop(&c, canvas.Point{X: 1, Y: 5}, 0.0, 1.0, 4.0, 5.0, s)

	get := func(row int) rune { return c.Cell(canvas.Point{X: 1, Y: 5 - row}).Rune }
	if get(0) != runes.LineDown { // frac(l)=0 < 0.5 → ╷
		t.Fatalf("bottom wick row: got %q want ╷\n%s", get(0), c.View())
	}
	if get(1) != runes.LowerBlockFour { // body bottom frac 0 < 0.5 → ▄ (line ╻)
		t.Fatalf("body bottom row: got %q want ▄\n%s", get(1), c.View())
	}
	for _, row := range []int{2, 3} {
		if get(row) != runes.FullBlock {
			t.Fatalf("body interior row %d: got %q want █\n%s", row, get(row), c.View())
		}
	}
	if get(4) != runes.LowerBlockFour { // body top frac 0 < 0.5 → ▄ (line ╻)
		t.Fatalf("body top row: got %q want ▄\n%s", get(4), c.View())
	}
	if get(5) != runes.LineDown { // top wick end frac 0 < 0.5 → ╷
		t.Fatalf("top wick row: got %q want ╷\n%s", get(5), c.View())
	}
}

// Same inputs, cell occupancy must match the line primitive exactly — the
// block style changes glyphs, never geometry.
func TestDrawCandlestickBlockOccupancyMatchesLine(t *testing.T) {
	s := lipgloss.NewStyle()
	a := canvas.New(3, 6)
	b := canvas.New(3, 6)
	DrawCandlestickBottomToTop(&a, canvas.Point{X: 1, Y: 5}, 0.4, 1.6, 3.2, 4.8, s)
	DrawCandlestickBlockBottomToTop(&b, canvas.Point{X: 1, Y: 5}, 0.4, 1.6, 3.2, 4.8, s)
	for row := 0; row <= 5; row++ {
		la := a.Cell(canvas.Point{X: 1, Y: 5 - row}).Rune != runes.Null
		lb := b.Cell(canvas.Point{X: 1, Y: 5 - row}).Rune != runes.Null
		if la != lb {
			t.Fatalf("occupancy mismatch at row %d: line=%v block=%v\nline:\n%s\nblock:\n%s",
				row, la, lb, a.View(), b.View())
		}
	}
}

// A body confined to one cell renders as a single full block.
func TestDrawCandlestickBlockSingleCellBody(t *testing.T) {
	s := lipgloss.NewStyle()
	c := canvas.New(3, 4)
	DrawCandlestickBlockBottomToTop(&c, canvas.Point{X: 1, Y: 3}, 1.0, 1.2, 1.8, 2.0, s)
	if r := c.Cell(canvas.Point{X: 1, Y: 2}).Rune; r != runes.FullBlock {
		t.Fatalf("single-cell body: got %q want █\n%s", r, c.View())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/evan/projects/ntcharts && go test ./canvas/graph/ -run TestDrawCandlestickBlock -count=1`
Expected: FAIL to build — `undefined: DrawCandlestickBlockBottomToTop` (and `runes.UpperHalfBlock` once referenced).

- [ ] **Step 3: Add the rune constant**

In `/Users/evan/projects/ntcharts/canvas/runes/runes.go`, next to `FullBlock` (line 45):

```go
	UpperHalfBlock  = '▀' // ▀
```

- [ ] **Step 4: Implement the primitive**

In `/Users/evan/projects/ntcharts/canvas/graph/graph.go`, after `DrawCandlestickBottomToTopWide`:

```go
// DrawCandlestickBlockBottomToTop draws a candle with a solid block body:
// interior body cells are FullBlock, the body's boundary cells use half
// blocks by the same half-rounding rule the line style applies (┃→█, ╻→▄,
// ╹→▀), and wick cells use the light line runes (│ ╵ ╷). Where a body
// boundary and a wick would share a cell, the block wins and the wick
// continues in the adjacent cell. A body confined to a single cell renders
// as one FullBlock. Value semantics of l, bl, bh, h match
// DrawCandlestickBottomToTop; coordinates (0,0) is top left of canvas.
func DrawCandlestickBlockBottomToTop(m *canvas.Model, p canvas.Point, l, bl, bh, h float64, s lipgloss.Style) {
	set := func(row int, r rune) {
		m.SetCell(canvas.Point{X: p.X, Y: p.Y - row}, canvas.NewCellWithStyle(r, s))
	}
	lf, blf := int(math.Floor(l)), int(math.Floor(bl))
	bhf, hf := int(math.Floor(bh)), int(math.Floor(h))

	// bottom wick — only rows below the body (the block wins shared cells)
	if lf < blf {
		lr := runes.LineUp
		if l-math.Floor(l) < 0.5 {
			lr = runes.LineDown
		}
		set(lf, lr)
		for i := lf + 1; i < blf; i++ {
			set(i, runes.LineVertical)
		}
	}

	// body
	if blf == bhf {
		set(blf, runes.FullBlock)
	} else {
		blr := runes.UpperHalfBlock // line style ╹
		if bl-math.Floor(bl) < 0.5 {
			blr = runes.LowerBlockFour // line style ╻ → ▄
		}
		set(blf, blr)
		for i := blf + 1; i < bhf; i++ {
			set(i, runes.FullBlock)
		}
		bhr := runes.LowerBlockFour // line style ╻ → ▄
		if bh-math.Floor(bh) >= 0.5 {
			bhr = runes.UpperHalfBlock // line style ╹ → ▀
		}
		set(bhf, bhr)
	}

	// top wick — only rows above the body
	if bhf < hf {
		hr := runes.LineDown
		if h-math.Floor(h) >= 0.5 {
			hr = runes.LineUp
		}
		for i := bhf + 1; i < hf; i++ {
			set(i, runes.LineVertical)
		}
		set(hf, hr)
	}
}
```

- [ ] **Step 5: Run tests to green, then the package suite**

Run: `go test ./canvas/graph/ -run TestDrawCandlestickBlock -count=1 -v` → PASS (3 tests).
Then: `go test ./canvas/... -count=1 && go vet ./canvas/... && gofmt -l canvas` → PASS/clean/empty.
If `TestDrawCandlestickBlockOccupancyMatchesLine` fails: the line primitive's junction handling may occupy a cell the block version skips (or vice versa) for these fractional inputs — fix the block version's row spans to match (the geometry contract governs), never the test.

- [ ] **Step 6: Commit**

```bash
git add canvas/runes/runes.go canvas/graph/graph.go canvas/graph/candlestick_block_test.go
git commit -m "feat(graph): DrawCandlestickBlockBottomToTop - solid block candle bodies"
```

---

### Task 2: `DrawCandleWithOpts` — opts struct, edge insets, style selection

**Files:**
- Modify: `/Users/evan/projects/ntcharts/linechart/timeserieslinechart/timeserieslinechart.go:343-470` (the `DrawCandleWidth`/`DrawCandle` pair)
- Test: `/Users/evan/projects/ntcharts/linechart/timeserieslinechart/candle_width_test.go` (extend)

**Interfaces:**
- Consumes: Task 1's `graph.DrawCandlestickBlockBottomToTop`; existing `graph.DrawCandlestickBottomToTop`.
- Produces: `type DrawCandleOpts struct { Width int; Block bool }` and `(m *Model) DrawCandleWithOpts(openName, highName, lowName, closeName string, bullStyle, bearStyle lipgloss.Style, opts DrawCandleOpts)`. `DrawCandleWidth(..., w)` → `DrawCandleWithOpts(..., DrawCandleOpts{Width: w})`; `DrawCandle(...)` → `{Width: 1}`. Task 3 calls `DrawCandleWithOpts`.

- [ ] **Step 1: Write the failing tests**

Append to `candle_width_test.go` (it already has `newCandleModel`/`pushCandles` helpers and a `stripAnsi`-equivalent if present — if not, reuse plain views: these tests use an unstyled `lipgloss.NewStyle()` so no ANSI is emitted):

```go
// Edge insets: with wide candles spanning the full time range, the first
// candle's body must be fully present immediately right of the axis and
// the last candle's body must end exactly at the final canvas column —
// no half-clipped edge candles.
func TestDrawCandleInsetsEdgeBodies(t *testing.T) {
	s := lipgloss.NewStyle()
	m := newCandleModel(t)
	const w = 5
	m.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: w})
	view := m.View()

	minCol := m.Origin().X + 1 // axis column + 1
	lastCol := m.Canvas.Width() - 1
	rows := strings.Split(view, "\n")
	fullLeft, fullRight := false, false
	for _, row := range rows {
		r := []rune(row)
		if len(r) <= lastCol {
			continue
		}
		leftRun, rightRun := 0, 0
		for x := minCol; x < minCol+w && x < len(r); x++ {
			if strings.ContainsRune("┃╽╿█▄▀", r[x]) {
				leftRun++
			}
		}
		for x := lastCol; x > lastCol-w && x >= 0; x-- {
			if strings.ContainsRune("┃╽╿█▄▀", r[x]) {
				rightRun++
			}
		}
		if leftRun == w {
			fullLeft = true
		}
		if rightRun == w {
			fullRight = true
		}
	}
	if !fullLeft {
		t.Fatalf("first candle body not fully inset right of the axis:\n%s", view)
	}
	if !fullRight {
		t.Fatalf("last candle body not fully inset against the right edge:\n%s", view)
	}
}

// Block style renders solid blocks; line style must not.
func TestDrawCandleBlockStyle(t *testing.T) {
	s := lipgloss.NewStyle()
	a := newCandleModel(t)
	a.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 3})
	b := newCandleModel(t)
	b.DrawCandleWithOpts("open", "high", "low", "close", s, s, DrawCandleOpts{Width: 3, Block: true})
	if strings.ContainsRune(a.View(), '█') {
		t.Fatal("line style must not contain full blocks")
	}
	if !strings.ContainsRune(b.View(), '█') {
		t.Fatalf("block style missing full blocks:\n%s", b.View())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./linechart/timeserieslinechart/ -run 'TestDrawCandleInsets|TestDrawCandleBlock' -count=1`
Expected: FAIL to build — `undefined: DrawCandleOpts` / no method `DrawCandleWithOpts`.

- [ ] **Step 3: Refactor into `DrawCandleWithOpts`**

Rename `DrawCandleWidth`'s body into the new method; the guards, dataset reads, `m.Clear()`/`m.DrawXYAxisAndLabel()`, bull/bear selection, `drawX` computation, and newest-candle clamp stay verbatim. Replace the per-candle column block (the `minCol`/`cw`/`left` section through the inner loop) with:

```go
		minCol := m.Origin().X
		if m.YStep() > 0 {
			minCol++ // Origin().X itself is the y-axis line column
		}
		cw := opts.Width
		if cw < 1 {
			cw = 1
		}
		// Edge inset: clamp the candle's CENTER so its whole body fits in
		// the drawable span — edge candles shift inward up to half a body
		// from their true time position (plot-inset behavior) instead of
		// rendering half-clipped. Skipped when the chart is narrower than
		// one candle (interval empty); the per-column guards below remain
		// the safety net for that and for dense overlapping data.
		half := (cw - 1) / 2
		right := cw - 1 - half
		if minCtr, maxCtr := minCol+half, m.Canvas.Width()-1-right; minCtr <= maxCtr {
			if drawX < minCtr {
				drawX = minCtr
			}
			if drawX > maxCtr {
				drawX = maxCtr
			}
		}
		left := drawX - half
		y := m.Origin().Y - 1
		for x := left; x < left+cw; x++ {
			if x < minCol {
				continue
			}
			q := canvas.Point{X: x, Y: y}
			if opts.Block {
				if x == drawX {
					graph.DrawCandlestickBlockBottomToTop(&m.Canvas, q, lData[i].Y, bl, bh, hData[i].Y, s)
				} else {
					graph.DrawCandlestickBlockBottomToTop(&m.Canvas, q, bl, bl, bh, bh, s)
				}
			} else {
				if x == drawX {
					graph.DrawCandlestickBottomToTop(&m.Canvas, q, lData[i].Y, bl, bh, hData[i].Y, s)
				} else {
					graph.DrawCandlestickBottomToTop(&m.Canvas, q, bl, bl, bh, bh, s)
				}
			}
		}
```

New type + delegations (the opts doc carries the inset note; extend `DrawCandleWithOpts`'s doc comment from `DrawCandleWidth`'s current one, adding the inset paragraph):

```go
// DrawCandleOpts configures candle drawing.
type DrawCandleOpts struct {
	// Width is the candle body width in columns; < 1 is treated as 1.
	Width int
	// Block renders solid block bodies (█ ▄ ▀) instead of line runes.
	Block bool
}

// DrawCandleWidth draws candles with the given body width using the
// default line style. See DrawCandleWithOpts.
func (m *Model) DrawCandleWidth(openName, highName, lowName, closeName string, bullStyle, bearStyle lipgloss.Style, width int) {
	m.DrawCandleWithOpts(openName, highName, lowName, closeName, bullStyle, bearStyle, DrawCandleOpts{Width: width})
}

// DrawCandle draws single-column line-style candles. See DrawCandleWithOpts.
func (m *Model) DrawCandle(openName, highName, lowName, closeName string, bullStyle, bearStyle lipgloss.Style) {
	m.DrawCandleWithOpts(openName, highName, lowName, closeName, bullStyle, bearStyle, DrawCandleOpts{Width: 1})
}
```

- [ ] **Step 4: Run new + existing tests**

Run: `go test ./linechart/timeserieslinechart/ -count=1 -v`
Expected: all PASS, including `TestDrawCandleDelegatesAtWidthOne` (width-1 unchanged: `half == right == 0` makes the inset exactly the pre-existing clamp) and `TestDrawCandleWidthPreservesAxisAndClampsNewest`. If the boundary test fails because the newest candle's body now ENDS at the last column instead of centering there, that is the intended inset — update that test's expectation and justify in the report; any other failure is a regression.

- [ ] **Step 5: Package suite + commit**

Run: `go test ./linechart/... -count=1 && go vet ./linechart/... && gofmt -l linechart` → PASS/clean/empty.

```bash
git add linechart/timeserieslinechart/timeserieslinechart.go linechart/timeserieslinechart/candle_width_test.go
git commit -m "feat(timeserieslinechart): DrawCandleWithOpts - edge insets + block style"
```

---

### Task 3: `Options.CandleStyle` in ntcharts-spec

**Files:**
- Modify: `/Users/evan/projects/ntcharts/spec/spec.go:249-263` (Options struct; constants near the Orientation constants at ~line 74)
- Modify: `/Users/evan/projects/ntcharts/spec/build.go` (`buildOHLC`, the `DrawCandleWidth` call at ~line 584)
- Modify: `/Users/evan/projects/ntcharts/spec/README.md` (Options documentation + OHLC section)
- Test: `/Users/evan/projects/ntcharts/spec/build_ohlc_test.go` (extend)

**Interfaces:**
- Consumes: Task 2's `DrawCandleWithOpts` + `DrawCandleOpts`.
- Produces: `Options.CandleStyle string` (JSON `candle_style`), constants `CandleStyleLine = "line"`, `CandleStyleBlock = "block"`.

- [ ] **Step 1: Write the failing tests**

Append to `spec/build_ohlc_test.go`:

```go
func TestBuildOHLCBlockStyle(t *testing.T) {
	s := ohlcSpec()
	s.Width = 60
	s.Options.CandleStyle = CandleStyleBlock
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	view := stripAnsi(got.(*timeserieslinechart.Model).View())
	if !strings.ContainsRune(view, '█') {
		t.Fatalf("candle_style block: no full blocks in view:\n%s", view)
	}

	s.Options.CandleStyle = CandleStyleLine
	got, err = Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsRune(stripAnsi(got.(*timeserieslinechart.Model).View()), '█') {
		t.Fatal("candle_style line must not render full blocks")
	}
}

func TestBuildOHLCUnknownCandleStyleErrors(t *testing.T) {
	s := ohlcSpec()
	s.Options.CandleStyle = "bogus"
	if _, err := Build(s); err == nil {
		t.Fatal("expected unknown candle_style error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./spec/ -run TestBuildOHLC -count=1`
Expected: FAIL to build — `undefined: CandleStyleBlock` / `s.Options.CandleStyle undefined`.

- [ ] **Step 3: Implement**

In `spec/spec.go`, next to the Orientation constants:

```go
// CandleStyle constants for Options.CandleStyle.
const (
	CandleStyleLine  = "line"
	CandleStyleBlock = "block"
)
```

In the `Options` struct, after `Stacked`:

```go
	// CandleStyle selects the OHLC candle rendering style: "" or
	// CandleStyleLine for box-drawing line runes, CandleStyleBlock for
	// solid block bodies. Unknown values are a Build error.
	CandleStyle string `json:"candle_style,omitempty"`
```

In `buildOHLC` (spec/build.go), before the options/model construction:

```go
	block := false
	switch s.Options.CandleStyle {
	case "", CandleStyleLine:
	case CandleStyleBlock:
		block = true
	default:
		return nil, fmt.Errorf("spec: unknown candle_style %q", s.Options.CandleStyle)
	}
```

and change the draw call:

```go
	m.DrawCandleWithOpts("open", "high", "low", "close", up, down,
		timeserieslinechart.DrawCandleOpts{
			Width: autoCandleWidth(m.GraphWidth(), times, tMin, tMax),
			Block: block,
		})
```

- [ ] **Step 4: Run to green, README, suite**

Run: `go test ./spec/ -run TestBuildOHLC -count=1 -v` → PASS.
Update `spec/README.md`: add `candle_style` to the Options documentation (values `"line"` (default) / `"block"`, OHLC-only, unknown values error) and mention it in the OHLC section (which already describes time-scaled placement + auto width; also note edge candles inset so whole bodies render).
Then: `go test ./... -count=1 && go vet ./... && gofmt -l canvas linechart spec` → all green/clean/empty.

- [ ] **Step 5: Commit**

```bash
git add spec/spec.go spec/build.go spec/build_ohlc_test.go spec/README.md
git commit -m "feat(spec): options.candle_style - line|block OHLC rendering"
```

---

### Task 4: flint template passthrough + demo + artifact train (flint-ntcharts)

**Files:**
- Modify: `/Users/evan/projects/flint-ntcharts/js/src/ntcharts/templates/candlestick.ts`
- Modify: `/Users/evan/projects/flint-ntcharts/js/src/ntcharts/types.ts` (options type gains `candle_style`)
- Modify: `/Users/evan/projects/flint-ntcharts/cmd/flint-edit/examples.go` (Candlestick example gains the chartProperty)
- Modify (generated): `compile/flint.wasm`, `compile/flint.wasm.buildinfo`
- Test: `/Users/evan/projects/flint-ntcharts/js/test/candlestick-style.test.ts` (create)

Work on flint-ntcharts branch `candle-block-style` (already created). ntcharts Tasks 1–3 are committed in the sibling, so the Go side understands `candle_style`.

**Interfaces:**
- Consumes: `assembleNtcharts` (js/src/ntcharts/index.js), the candlestick fixture at `testdata/fixtures-terminal/candlestick.json` as a template for inline test input.
- Produces: `options.candle_style` in emitted specs when `chart_spec.chartProperties.candleStyle` is `"line"` or `"block"`.

- [ ] **Step 1: Write the failing vitest**

Create `js/test/candlestick-style.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { assembleNtcharts } from "../src/ntcharts/index.js";

const fixturePath = fileURLToPath(
  new URL("../../testdata/fixtures-terminal/candlestick.json", import.meta.url));

function candlestickInput(candleStyle?: string) {
  const input = JSON.parse(readFileSync(fixturePath, "utf8"));
  if (candleStyle !== undefined) {
    input.chart_spec.chartProperties = { candleStyle };
  }
  return input;
}

describe("candlestick candleStyle passthrough", () => {
  it("emits options.candle_style for 'block'", () => {
    const out = assembleNtcharts(candlestickInput("block"));
    expect(out.options?.candle_style).toBe("block");
  });
  it("emits options.candle_style for 'line'", () => {
    const out = assembleNtcharts(candlestickInput("line"));
    expect(out.options?.candle_style).toBe("line");
  });
  it("ignores unknown values and absence", () => {
    expect(assembleNtcharts(candlestickInput("fancy")).options?.candle_style).toBeUndefined();
    expect(assembleNtcharts(candlestickInput()).options?.candle_style).toBeUndefined();
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd js && npx vitest run test/candlestick-style.test.ts`
Expected: FAIL — `options?.candle_style` is undefined for "block" (no passthrough exists).

- [ ] **Step 3: Implement the passthrough**

In `js/src/ntcharts/types.ts`, add to the options type (find the type used by `emit.options` — it carries `show_legend`, `stacked`, `orientation`):

```ts
  candle_style?: "line" | "block";
```

In `js/src/ntcharts/templates/candlestick.ts`, inside `instantiate` after the `emit.theme` line:

```ts
    // chartProperties passes through unvalidated in this backend (see the
    // "Deviations" note in ../shims.ts); candleStyle is the one property
    // this template honors: "line" | "block" map to ntcharts-spec
    // options.candle_style, anything else is silently ignored.
    const candleStyle = _spec?.chartProperties?.candleStyle;
    if (candleStyle === "line" || candleStyle === "block") {
      emit.options = { ...emit.options, candle_style: candleStyle };
    }
```

First verify the property path: `instantiate`'s first argument is the chart spec object. Check with `grep -n "chartProperties" js/node_modules/flint-chart/dist/index.d.ts | head -3` and, if ambiguous, add a temporary `console.log(Object.keys(_spec))` while running the vitest — the object containing `chartType`/`encodings` is the right one; `chartProperties` is its optional sibling key. Remove any temporary logging before committing.

- [ ] **Step 4: Run vitest + typecheck to green**

Run: `cd js && npx tsc --noEmit && npx vitest run`
Expected: all green, including the drift check — no golden changes (`git status --short testdata/` must be empty; if any golden changed, STOP and report BLOCKED).

- [ ] **Step 5: Update the demo example**

In `cmd/flint-edit/examples.go`, in the Candlestick example's `chart_spec`, add the property (after `"chartType": "Candlestick Chart",`):

```json
    "chartProperties": { "candleStyle": "block" },
```

- [ ] **Step 6: Rebuild wasm + full verification**

Run: `make wasm` (regenerates `compile/flint.wasm` + `.buildinfo`; `flint-chart: 0.2.1` stays, hashes change).
Then: `cd js && npm run gen-expected` and verify `git status --short testdata/` is STILL empty (fixtures don't use the property — any diff is BLOCKED).
Then: `go test ./... -count=1 && go vet ./...` in flint-ntcharts → all 6 packages PASS (flint-edit's `TestExamplesCompile` now compiles the block-style example through the new wasm against the new sibling renderer).

- [ ] **Step 7: Commit**

```bash
git add js/src/ntcharts/templates/candlestick.ts js/src/ntcharts/types.ts js/test/candlestick-style.test.ts cmd/flint-edit/examples.go compile/flint.wasm compile/flint.wasm.buildinfo
git commit -m "feat(ts): candleStyle chartProperty -> options.candle_style; block-style demo"
```
