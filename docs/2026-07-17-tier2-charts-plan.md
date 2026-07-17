# Tier-2 Charts Implementation Plan (flint → ntcharts, Phase 7 — final)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Candlestick (OHLC) and Sparkline render end-to-end: ntcharts-spec `Build()` implementations (the last two scaffolds), flint TS templates (`"Candlestick Chart"`, `"Sparkline"`), fixtures/goldens/speccheck cascade, wasm + browser-bundle rebuild, re-vendor, docs. This completes the design's chart roadmap.

**Architecture:** `buildSparkline` maps a single series onto the existing `sparkline` package (Push/Draw). `buildOHLC` has no high-level model to target — it owns scaling and draws onto a raw `canvas.Model` with `graph.DrawXYAxis` + `graph.DrawCandlestickBottomToTop` (heights are FRACTIONAL ROWS drawn upward from an origin; ½-row rune resolution). Up/down candle colors follow a documented palette convention. The TS templates emit `Series.OHLC` (ms timestamps, time-sorted) and single-series sparkline values. Two new terminal fixtures ride the whole verification train: goldens → Node references → wasm parity (13) → speccheck (13) → browser bundle re-vendor.

**Tech Stack:** existing (no new deps anywhere).

## Global Constraints

- Repos: **ntcharts** `/Users/evan/projects/ntcharts` branch `spec` (Tasks 1-2; NEVER `git add -A` — four go.mod/go.sum files carry uncommitted user changes); **flint-ntcharts** branch main (Task 3-4); **booba-shim** branch `flintchart` (Task 4 re-vendor only; ci.yml untouched as always).
- Verified ntcharts API (do not re-derive): `graph.DrawCandlestickBottomToTop(m *canvas.Model, p canvas.Point, l, bl, bh, h float64, s lipgloss.Style)` — l/bl/bh/h are candle low / body-low / body-high / high as HEIGHTS IN ROWS going up from p (fractional ok); assumes h≥bh, l≤bl; `graph.DrawXYAxis(m *canvas.Model, p canvas.Point, s lipgloss.Style)` draws axes from an origin (candles start at `p.Add(Point{X:1+i, Y:-1})` per `examples/graph/candlesticks/main.go:71-81`); `sparkline.New(w, h int, opts ...Option) Model`, `(*Model).Push(f float64)`, `(*Model).Draw()`, `sparkline.WithMaxValue(f float64)`, `sparkline.WithStyle(s lipgloss.Style)`. Verify `canvas.New`'s exact signature in `canvas/canvas.go` before use (bounded adaptation; record it).
- **OHLC semantics (frozen for the fidelity matrix):** price range = [min(L), max(H)] with the one-sided YAxis pin rule (reuse `resolveYRange`'s semantics — inverted range errors); candles ordered by T ascending; when candle count exceeds usable columns, keep the MOST RECENT ones and emit no error (ntcharts-spec has no warning channel — document the clamp in the README fidelity matrix; the flint TS side is where a truncation warning can surface, via flint's own overflow machinery or a template warning); up candle (C ≥ O) colored `Theme.Palette[0]` (default `#26a69a`), down candle `Theme.Palette[1]` (default `#ef5350`) — palette-slot convention documented in both READMEs.
- **Sparkline semantics:** exactly one series (else error — mirrors the stacked-bar honesty gate); `YAxis.Max` pin → `WithMaxValue`; `YAxis.Min` unsupported by the sparkline package → documented as ignored in the fidelity matrix; values pushed in order.
- Golden/Example hygiene from prior phases applies: ANSI-free goldens (color-free specs in Examples; note OHLC ALWAYS styles candles — use the `%T`-pattern Example like heatmap, NOT a View() golden), `// Output:` capture procedure, TrimRight for canvas-backed views if needed.
- New fixtures go in `testdata/fixtures-terminal/` ONLY (pixel parity dir stays 5); parity/speccheck guards move ≥11→≥13.
- Task-4 sequencing (P6 lesson): commit ALL flint-ntcharts source changes FIRST, then `make wasm && make browser-shim` from the clean tree, then commit artifacts, then re-vendor booba-shim. PROVENANCE must record the artifact commit with no `-dirty` suffix; booba-shim `vendor_test` must pass post-vendor.
- All suites green at each boundary in whichever repos a task touched.

---

### Task 1: ntcharts — buildSparkline

**Files (ntcharts, branch spec):**
- Modify: `spec/build.go` (route + implement)
- Create: `spec/build_sparkline_test.go`
- Modify: `spec/example_test.go` (ExampleBuild_sparkline), `spec/README.md` (status row → full; fidelity rows)

**Interfaces:**
- Produces `buildSparkline(s Spec) (any, error)` → `*sparkline.Model`.

- [ ] **Step 1: Failing tests** — `spec/build_sparkline_test.go`:

```go
package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/sparkline"
)

func sparkSpec() Spec {
	return Spec{
		Type: ChartTypeSparkline, Width: 30, Height: 4,
		Data: Data{Series: []Series{{Name: "cpu", Values: []DataPoint{
			{Y: 1}, {Y: 4}, {Y: 2}, {Y: 7}, {Y: 5}, {Y: 9}, {Y: 3},
		}}}},
	}
}

func TestBuildSparkline(t *testing.T) {
	got, err := Build(sparkSpec())
	if err != nil {
		t.Fatalf("Build(sparkline): %v", err)
	}
	m, ok := got.(*sparkline.Model)
	if !ok {
		t.Fatalf("Build(sparkline) returned %T, want *sparkline.Model", got)
	}
	if strings.TrimSpace(m.View()) == "" {
		t.Fatal("sparkline view is empty")
	}
}

func TestBuildSparklineSingleSeriesOnly(t *testing.T) {
	s := sparkSpec()
	s.Data.Series = append(s.Data.Series, Series{Name: "b", Values: []DataPoint{{Y: 1}}})
	if _, err := Build(s); err == nil || !strings.Contains(err.Error(), "one series") {
		t.Fatalf("expected single-series error, got %v", err)
	}
}

func TestBuildSparklineMaxPin(t *testing.T) {
	s := sparkSpec()
	s.YAxis.Max = f64(20)
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(sparkline, max pin): %v", err)
	}
}
```

RED (`not yet implemented`) →

- [ ] **Step 2: Implement**

```go
// buildSparkline renders a single series onto a sparkline model. The
// sparkline package is single-series and has no Y-minimum: multiple series
// error (like grouped bars), YAxis.Min is documented-ignored, YAxis.Max maps
// to WithMaxValue.
func buildSparkline(s Spec) (any, error) {
	if len(s.Data.Series) != 1 {
		return nil, fmt.Errorf("spec: sparkline supports exactly one series; got %d", len(s.Data.Series))
	}
	ser := s.Data.Series[0]
	opts := []sparkline.Option{sparkline.WithStyle(seriesStyle(ser, 0, s.Theme))}
	if s.YAxis.Max != nil {
		opts = append(opts, sparkline.WithMaxValue(*s.YAxis.Max))
	}
	m := sparkline.New(s.Width, s.Height, opts...)
	for _, p := range ser.Values {
		m.Push(p.Y)
	}
	m.Draw()
	return &m, nil
}
```

Route `case ChartTypeSparkline: return buildSparkline(s)`.

- [ ] **Step 3: Golden Example + docs + verify + commit** — ExampleBuild_sparkline (color-free spec; View() golden is fine — sparkline without a style renders plain block runes; verify `grep -c $'\x1b' spec/example_test.go` = 0, TrimRight if the view pads). README: status row full (terminal; web stays scaffold), fidelity rows (single-series error, Min ignored, Max→WithMaxValue).

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -v && go vet ./spec/
git add spec/build.go spec/build_sparkline_test.go spec/example_test.go spec/README.md
git commit -m "feat(spec): Build() sparklines"
```

---

### Task 2: ntcharts — buildOHLC

**Files (ntcharts, branch spec):**
- Modify: `spec/build.go`
- Create: `spec/build_ohlc_test.go`
- Modify: `spec/example_test.go` (ExampleBuild_ohlc — %T pattern, candles are always styled), `spec/README.md` (status + fidelity: palette up/down slots, most-recent clamp, one-sided pins)

**Interfaces:**
- Produces `buildOHLC(s Spec) (any, error)` → `*canvas.Model` (a new concrete return type — document in build.go's type table comment).

- [ ] **Step 1: Failing tests** — `spec/build_ohlc_test.go`:

```go
package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
)

func ohlcSpec() Spec {
	return Spec{
		Type: ChartTypeOHLC, Width: 40, Height: 12,
		Data: Data{Series: []Series{{Name: "px", OHLC: []OHLCPoint{
			{T: "2026-01-05", O: 100, H: 108, L: 97, C: 105},
			{T: "2026-01-06", O: 105, H: 112, L: 103, C: 110},
			{T: "2026-01-07", O: 110, H: 111, L: 98, C: 99},
			{T: "2026-01-08", O: 99, H: 106, L: 96, C: 104},
		}}}},
	}
}

func TestBuildOHLC(t *testing.T) {
	got, err := Build(ohlcSpec())
	if err != nil {
		t.Fatalf("Build(ohlc): %v", err)
	}
	m, ok := got.(*canvas.Model)
	if !ok {
		t.Fatalf("Build(ohlc) returned %T, want *canvas.Model", got)
	}
	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("ohlc view is empty")
	}
	// candle runes are box-drawing verticals; assert SOMETHING beyond axis was drawn
	if !strings.ContainsAny(view, "│┃╽╿") {
		t.Fatalf("no candle runes found in view:\n%s", view)
	}
}

func TestBuildOHLCClampsToMostRecent(t *testing.T) {
	s := ohlcSpec()
	// widen the data far beyond usable columns of a narrow chart
	s.Width = 12
	pts := s.Data.Series[0].OHLC
	for i := 0; i < 40; i++ {
		pts = append(pts, OHLCPoint{T: "2026-02-01", O: 100, H: 101, L: 99, C: 100})
	}
	s.Data.Series[0].OHLC = pts
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(ohlc, clamp): %v", err)
	}
}

func TestBuildOHLCInvertedPinErrors(t *testing.T) {
	s := ohlcSpec()
	s.YAxis.Min = f64(500) // above all highs
	if _, err := Build(s); err == nil {
		t.Fatal("expected inverted-range error")
	}
}
```

RED →

- [ ] **Step 2: Implement**

```go
// Default up/down candle colors (overridden by Theme.Palette slots 0/1).
const (
	defaultUpColor   = "#26a69a"
	defaultDownColor = "#ef5350"
)

// buildOHLC renders candlesticks onto a raw canvas. ntcharts has no
// high-level OHLC model; this surface owns the price scaling and layout.
// Palette convention: Theme.Palette[0] = up candles, [1] = down candles.
// When there are more candles than columns, the most recent ones are kept.
func buildOHLC(s Spec) (any, error) {
	var pts []OHLCPoint
	for _, ser := range s.Data.Series {
		if len(ser.OHLC) > 0 {
			pts = ser.OHLC
			break
		}
	}
	if len(pts) == 0 {
		return nil, fmt.Errorf("spec: ohlc requires Series.OHLC points")
	}

	minP, maxP := math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		minP = math.Min(minP, p.L)
		maxP = math.Max(maxP, p.H)
	}
	if s.YAxis.Min != nil {
		minP = *s.YAxis.Min
	}
	if s.YAxis.Max != nil {
		maxP = *s.YAxis.Max
	}
	if minP > maxP {
		return nil, fmt.Errorf("spec: y_axis min %v exceeds max %v", minP, maxP)
	}
	if minP == maxP {
		minP, maxP = minP-1, maxP+1
	}

	c := canvas.New(s.Width, s.Height) // adapt to the actual constructor signature
	origin := canvas.Point{X: 0, Y: s.Height - 1}
	graph.DrawXYAxis(&c, origin, seriesAxisStyle(s.Theme))

	usableCols := s.Width - 2  // axis column + margin
	usableRows := s.Height - 2 // axis row + headroom
	if usableCols < 1 || usableRows < 1 {
		return nil, fmt.Errorf("spec: ohlc needs at least 3x3 cells; got %dx%d", s.Width, s.Height)
	}
	if len(pts) > usableCols {
		pts = pts[len(pts)-usableCols:] // most recent candles win (documented)
	}

	up := candleStyle(s.Theme, 0, defaultUpColor)
	down := candleStyle(s.Theme, 1, defaultDownColor)
	scale := func(v float64) float64 {
		return (v - minP) / (maxP - minP) * float64(usableRows)
	}
	for i, p := range pts {
		st := up
		if p.C < p.O {
			st = down
		}
		bl, bh := math.Min(p.O, p.C), math.Max(p.O, p.C)
		graph.DrawCandlestickBottomToTop(&c,
			canvas.Point{X: origin.X + 1 + i, Y: origin.Y - 1},
			scale(p.L), scale(bl), scale(bh), scale(p.H), st)
	}
	return &c, nil
}

func candleStyle(t Theme, slot int, fallback string) lipgloss.Style {
	color := fallback
	if slot < len(t.Palette) && t.Palette[slot] != "" {
		color = t.Palette[slot]
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color))
}

func seriesAxisStyle(t Theme) lipgloss.Style {
	if t.Foreground != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(t.Foreground))
	}
	return lipgloss.NewStyle()
}
```

Route the case; add `github.com/NimbleMarkets/ntcharts/v2/canvas` + `.../canvas/graph` imports. Clamp candle values to the pinned range before scaling if pins are tighter than data (values outside [minP,maxP] must not draw outside the canvas — clamp scale() output to [0, usableRows]). Adapt `canvas.New` and origin conventions per what `examples/graph/candlesticks/main.go` actually constructs — verify the axis + candle placement visually via the test's view dump before finalizing.

- [ ] **Step 3: Example (%T pattern) + docs + verify + commit**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -v && go vet ./spec/ && grep -c $'\x1b' spec/example_test.go   # must stay 0
git add spec/build.go spec/build_ohlc_test.go spec/example_test.go spec/README.md
git commit -m "feat(spec): Build() OHLC candlesticks via canvas primitives"
```

---

### Task 3: flint-ntcharts — TS templates + fixtures + verification train

**Files (flint-ntcharts, main):**
- Create: `js/src/ntcharts/templates/candlestick.ts`, `js/src/ntcharts/templates/sparkline.ts`
- Modify: `js/src/ntcharts/templates/index.ts`, `js/src/ntcharts/types.ts` (only if a gap emerges — record)
- Create: `testdata/fixtures-terminal/candlestick.json`, `testdata/fixtures-terminal/sparkline.json`
- Modify: `js/test/golden-terminal.test.ts` (2 cases), `speccheck/speccheck_test.go` + `compile/compile_test.go` (guards ≥13), `README.md` (supported types 7)
- Generated: 2 terminal goldens, 2 expected-terminal references, rebuilt `compile/flint.wasm` + buildinfo

**Interfaces:**
- `"Candlestick Chart"` template: channels `["x", "open", "high", "low", "close"]`, emits `type: "ohlc"`, single `Series{Name: <x field or "ohlc">, OHLC: [{t: msNumber, o, h, l, c}]}` sorted by t ascending (reuse the line template's toMs; drop NaN-t points with the `invalid-temporal-x`-style warning); `x_axis: {type: "time", format}`; theme palette `["#26a69a", "#ef5350"]` (the documented up/down slots); YAxis left to autoscale unless flint pinned a domain.
- `"Sparkline"` template: channels `["x", "y"]`, emits `type: "sparkline"`, single series of y values (sorted by x when x present), no axes.

- [ ] **Step 1: Fixtures + failing golden tests**

`testdata/fixtures-terminal/candlestick.json`:

```json
{
  "data": { "values": [
    {"date": "2026-01-05", "open": 100, "high": 108, "low": 97,  "close": 105},
    {"date": "2026-01-06", "open": 105, "high": 112, "low": 103, "close": 110},
    {"date": "2026-01-07", "open": 110, "high": 111, "low": 98,  "close": 99},
    {"date": "2026-01-08", "open": 99,  "high": 106, "low": 96,  "close": 104},
    {"date": "2026-01-09", "open": 104, "high": 115, "low": 102, "close": 114}
  ]},
  "chart_spec": {
    "chartType": "Candlestick Chart",
    "encodings": {
      "x": {"field": "date"},
      "open": {"field": "open"}, "high": {"field": "high"},
      "low": {"field": "low"}, "close": {"field": "close"}
    }
  }
}
```

`testdata/fixtures-terminal/sparkline.json`:

```json
{
  "data": { "values": [
    {"t": 1, "v": 3}, {"t": 2, "v": 5}, {"t": 3, "v": 2}, {"t": 4, "v": 8},
    {"t": 5, "v": 6}, {"t": 6, "v": 9}, {"t": 7, "v": 4}, {"t": 8, "v": 7}
  ]},
  "chart_spec": {
    "chartType": "Sparkline",
    "encodings": {"x": {"field": "t"}, "y": {"field": "v"}}
  }
}
```

Golden tests (append to `js/test/golden-terminal.test.ts`):

```ts
describe("golden-terminal: candlestick", () => {
  it("emits time-sorted ohlc points with the up/down palette", () => {
    const out = assembleTerminalFixture("candlestick");
    expect(out.type).toBe("ohlc");
    const pts = out.data.series[0].ohlc!;
    expect(pts).toHaveLength(5);
    const ts = pts.map((p: any) => p.t as number);
    expect(ts).toEqual([...ts].sort((a, b) => a - b));
    expect(pts[0]).toMatchObject({ o: 100, h: 108, l: 97, c: 105 });
    expect(out.theme?.palette).toEqual(["#26a69a", "#ef5350"]);
    expect(out.x_axis?.format?.kind).toBe("time");
    expectTerminalGolden("candlestick", out);
  });
});

describe("golden-terminal: sparkline", () => {
  it("emits a single y-value series", () => {
    const out = assembleTerminalFixture("sparkline");
    expect(out.type).toBe("sparkline");
    expect(out.data.series).toHaveLength(1);
    expect(out.data.series[0].values!.map((p: any) => p.y)).toEqual([3, 5, 2, 8, 6, 9, 4, 7]);
    expectTerminalGolden("sparkline", out);
  });
});
```

(Adapt helper names to what golden-terminal.test.ts actually uses.) Verify the flint chart-type names first: `node -e "import('flint-chart').then(m => console.log(m.vlAllTemplateDefs.filter(d => /Candle|Spark/.test(d.chart)).map(d => [d.chart, d.channels])))"` — confirm `"Candlestick Chart"` channel names are exactly open/high/low/close and `"Sparkline"`'s are x/y; adapt if not (record).

- [ ] **Step 2: Implement templates** — follow the established template structure (line.ts is the closest sibling for candlestick's temporal handling; both templates split nothing — single series). Candlestick instantiate: read `cs.open.field` etc. from channelSemantics; rows → `{t: toMs(row[xField]), o, h, l, c: Number(...)}`; sort by t; NaN-t drop + one warning. Sparkline instantiate: y values via `Number(row[yField])`, sorted by x when `cs.x` exists.

- [ ] **Step 3: The verification train**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npx vitest run && npx tsc --noEmit           # goldens captured; inspect both files
npm run gen-expected                          # references now 13
cd .. && make wasm                            # bundle changed → rebuild
# update parity guard ≥11 → ≥13 in compile/compile_test.go, speccheck guard likewise
go test ./... -count=1                        # parity 13/13, speccheck 13/13 (Go Build of the new goldens
                                              # exercises Task 1-2's buildSparkline/buildOHLC end-to-end)
```

READ both goldens before committing; paste into the report. If speccheck fails on the new goldens, the mismatch is between the TS emission and the Go Build contract — fix on whichever side violates THIS PLAN's frozen semantics (do not weaken semantics silently; report the divergence).

- [ ] **Step 4: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add js/src/ntcharts/templates/ js/test/golden-terminal.test.ts testdata/fixtures-terminal/ testdata/ntspec-golden-terminal/ testdata/expected-terminal/ compile/flint.wasm compile/flint.wasm.buildinfo compile/compile_test.go speccheck/speccheck_test.go README.md
git commit -m "feat(ts): Candlestick + Sparkline templates; verification train to 13 fixtures"
```

---

### Task 4: browser re-vendor + cross-repo docs closeout

**Files:**
- flint-ntcharts: none beyond Task 3 (browser bundle is built from the committed tree)
- booba-shim (branch flintchart): `web/flintchart/flintchart-shim.js` + `PROVENANCE.txt` + both assets-mirror copies (re-vendored), `flintchart/README.md` (supported types note), `CHANGELOG.md` (bundle update line)

**Interfaces:**
- The vendored browser bundle gains the two new chart types; provenance records the Task-3 commit, clean tree; `vendor_test` green.

- [ ] **Step 1: Rebuild from the clean tree** (flint-ntcharts HEAD = Task 3's commit; `git status --porcelain` must be empty of tracked changes):

```bash
cd /Users/evan/projects/flint-ntcharts && make browser-shim
cd js && npm run smoke:browser    # smoke still green with the new bundle
```

PROVENANCE must show the Task-3 commit with NO -dirty suffix.

- [ ] **Step 2: Re-vendor + verify**

```bash
cd /Users/evan/projects/booba-shim   # branch flintchart
cp /Users/evan/projects/flint-ntcharts/js/dist/flintchart-shim.mjs web/flintchart/flintchart-shim.js
cp /Users/evan/projects/flint-ntcharts/js/dist/flintchart-shim.PROVENANCE.txt web/flintchart/PROVENANCE.txt
cp web/flintchart/flintchart-shim.js cmd/booba-shim-assets/assets/flintchart/flintchart-shim.js
cp web/flintchart/PROVENANCE.txt cmd/booba-shim-assets/assets/flintchart/PROVENANCE.txt
go test ./... && go vet ./... && GOOS=js GOARCH=wasm go build ./...   # vendor_test revalidates the chain
node examples/flintchart-compile/smoke.mjs                            # example still green
```

Headless proof the new types reached the browser bundle: a one-off node check compiling the candlestick fixture through the VENDORED file (adapt smoke.mjs's pattern) — output envelope must have `spec.type === "ohlc"`.

- [ ] **Step 3: Docs + commits**

- booba-shim: CHANGELOG line under the flintchart Unreleased entry ("bundle updated: adds Candlestick/Sparkline"); flintchart/README.md supported-types sentence. Commit (explicit paths; ci.yml untouched): `chore(flintchart): re-vendor bundle with tier-2 chart types`
- ntcharts README + flint-ntcharts README should already be done in Tasks 1-3 — verify both list 7 types consistently; fix inline if not (record).

- [ ] **Step 4: Final full sweep — all three repos**

```bash
cd /Users/evan/projects/ntcharts && go test ./spec/ -count=1 && go vet ./spec/
cd /Users/evan/projects/flint-ntcharts && go test ./... -count=1 && (cd js && npx vitest run && npx tsc --noEmit && npm run build)
cd /Users/evan/projects/booba-shim && go test ./... && GOOS=js GOARCH=wasm go build ./...
```

All green = the design's chart roadmap is complete.
