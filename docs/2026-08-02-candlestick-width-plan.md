# Candlestick Width & Time-Scaling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Candlestick charts spread across the full x axis at time-scaled positions with labeled axes and auto-width multi-column bodies.

**Architecture:** Task 1 adds a width-composing graph primitive (`DrawCandlestickBottomToTopWide`: center column = full candle, side columns = body-only candles). Task 2 threads width through `timeserieslinechart` (`DrawCandleWidth`; `DrawCandle` delegates at width 1). Task 3 rebases `spec.buildOHLC` onto `timeserieslinechart` with the same option wiring as `buildTimeSeries`, an auto-width heuristic, a date-only extension to `pointTime`, and updated docs/tests.

**Tech Stack:** Go, ntcharts (canvas/graph, linechart/timeserieslinechart, spec).

**Spec:** `docs/2026-08-02-candlestick-width-design.md` (in flint-ntcharts)

## Global Constraints

- ALL code changes are in `/Users/evan/projects/ntcharts`, branch `spec` (already checked out — never switch, pull, or push). flint-ntcharts gets zero code changes.
- Do not stage the untracked `examples/combo_chart.html`. Stage explicit paths only — no `git add -A`, no `git add .`, no `git commit -a`.
- `DrawCandle`'s output for existing callers must stay byte-identical (it becomes a width-1 delegation).
- Auto width: `max(1, graphWidth/n − 1)`, clamped to `minAdjacentGap − 1` between time-scaled centers (skip clamp when n == 1 or all times equal), capped at 7, forced odd (decrement if even), floor 1.
- Verified API facts: `timeserieslinechart.TimePoint{Time time.Time, Value float64}`; `PushDataSet(name string, t TimePoint)`; `pointTime(x any) (time.Time, bool)` in `spec/helpers.go` accepts time.Time / RFC3339(Nano) string / ms-since-epoch numerics; `candleStyle(theme, slot, fallback)` and `seriesAxisStyle` exist in `spec/build.go`; `makeTimeAxisFormatter(layout)` at `spec/build.go:380`; the model's `View()` is ready after the draw call (no separate DrawAll needed for the candle path — `DrawCandleWidth` itself clears and draws axes+labels).
- Each task ends green: `go test ./canvas/... ./linechart/... ./spec/ -count=1`, `go vet ./...` clean, `gofmt -l canvas linechart spec` empty (all in ntcharts).
- Existing OHLC spec-test data uses date-only `T: "2026-01-05"` strings; Task 3 extends `pointTime` with the `"2006-01-02"` layout so they parse (deliberate scope, benefits hand-authored specs).

---

### Task 1: `DrawCandlestickBottomToTopWide` (canvas/graph)

**Files:**
- Modify: `/Users/evan/projects/ntcharts/canvas/graph/graph.go` (add function after `DrawCandlestickBottomToTop`, which ends near line 897)
- Test: Create `/Users/evan/projects/ntcharts/canvas/graph/candlestick_wide_test.go`

**Interfaces:**
- Consumes: existing `DrawCandlestickBottomToTop(m *canvas.Model, p canvas.Point, l, bl, bh, h float64, s lipgloss.Style)`.
- Produces: `DrawCandlestickBottomToTopWide(m *canvas.Model, p canvas.Point, width int, l, bl, bh, h float64, s lipgloss.Style)` — Task 2 calls this.

- [ ] **Step 1: Write the failing test**

Create `/Users/evan/projects/ntcharts/canvas/graph/candlestick_wide_test.go`:

```go
// ntcharts - Copyright (c) 2026 Neomantra Corp.

package graph

import (
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/canvas/runes"

	"charm.land/lipgloss/v2"
)

// Width 1 must be exactly the single-column primitive.
func TestDrawCandlestickWideWidthOneIdentical(t *testing.T) {
	s := lipgloss.NewStyle()
	a := canvas.New(5, 5)
	b := canvas.New(5, 5)
	DrawCandlestickBottomToTop(&a, canvas.Point{X: 2, Y: 4}, 0.0, 1.0, 3.0, 4.0, s)
	DrawCandlestickBottomToTopWide(&b, canvas.Point{X: 2, Y: 4}, 1, 0.0, 1.0, 3.0, 4.0, s)
	if a.View() != b.View() {
		t.Fatalf("width-1 wide candle differs from single-column primitive:\n%q\nvs\n%q", a.View(), b.View())
	}
}

// Width 3: side columns carry body rows only; wick rows stay empty there.
func TestDrawCandlestickWideThreeColumns(t *testing.T) {
	s := lipgloss.NewStyle()
	c := canvas.New(5, 5)
	// l=0 → row 4 (lower wick), body 1..3 → rows 3..1, h=4 → row 0 (upper wick)
	DrawCandlestickBottomToTopWide(&c, canvas.Point{X: 2, Y: 4}, 3, 0.0, 1.0, 3.0, 4.0, s)

	for _, x := range []int{1, 3} { // side columns
		for _, y := range []int{1, 2, 3} { // body rows
			if c.Cell(canvas.Point{X: x, Y: y}).Rune == runes.Null {
				t.Fatalf("side column %d missing body rune at row %d:\n%s", x, y, c.View())
			}
		}
		for _, y := range []int{0, 4} { // pure wick rows
			if c.Cell(canvas.Point{X: x, Y: y}).Rune != runes.Null {
				t.Fatalf("side column %d must not draw wick row %d:\n%s", x, y, c.View())
			}
		}
	}
	for y := 0; y <= 4; y++ { // center column: full candle
		if c.Cell(canvas.Point{X: 2, Y: y}).Rune == runes.Null {
			t.Fatalf("center column missing rune at row %d:\n%s", y, c.View())
		}
	}
}

// Even width renders as next-lower-odd plus one extra column on the right.
func TestDrawCandlestickWideEvenWidth(t *testing.T) {
	s := lipgloss.NewStyle()
	c := canvas.New(6, 5)
	DrawCandlestickBottomToTopWide(&c, canvas.Point{X: 2, Y: 4}, 4, 0.0, 1.0, 3.0, 4.0, s)
	// columns 1,2,3,4 covered; column 0 and 5 empty at body row 2
	for _, x := range []int{1, 2, 3, 4} {
		if c.Cell(canvas.Point{X: x, Y: 2}).Rune == runes.Null {
			t.Fatalf("column %d missing body rune for even width:\n%s", x, c.View())
		}
	}
	for _, x := range []int{0, 5} {
		if c.Cell(canvas.Point{X: x, Y: 2}).Rune != runes.Null {
			t.Fatalf("column %d must be empty for width 4 centered at 2:\n%s", x, c.View())
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/evan/projects/ntcharts && go test ./canvas/graph/ -run TestDrawCandlestickWide -count=1`
Expected: FAIL to build — `undefined: DrawCandlestickBottomToTopWide`

- [ ] **Step 3: Implement the primitive**

In `/Users/evan/projects/ntcharts/canvas/graph/graph.go`, directly after `DrawCandlestickBottomToTop`'s closing brace:

```go
// DrawCandlestickBottomToTopWide draws a candle whose body spans `width`
// columns centered on p.X; the wick renders only in the center column.
// Side columns are drawn as body-only candles (l=bl, h=bh), so the body
// runes compose from the existing single-column primitive. width < 1 is
// treated as 1; width 1 is exactly DrawCandlestickBottomToTop. Even widths
// render as the next lower odd width plus one extra column on the right.
// Columns outside the canvas are clipped by the underlying rune drawing.
func DrawCandlestickBottomToTopWide(m *canvas.Model, p canvas.Point, width int, l, bl, bh, h float64, s lipgloss.Style) {
	if width < 1 {
		width = 1
	}
	left := p.X - (width-1)/2
	for x := left; x < left+width; x++ {
		q := canvas.Point{X: x, Y: p.Y}
		if x == p.X {
			DrawCandlestickBottomToTop(m, q, l, bl, bh, h, s)
		} else {
			DrawCandlestickBottomToTop(m, q, bl, bl, bh, bh, s)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./canvas/graph/ -run TestDrawCandlestickWide -count=1 -v`
Expected: PASS (3 tests). If `TestDrawCandlestickWideThreeColumns` fails on a side-column wick row being non-null, inspect: the body-only call `(bl, bl, bh, bh)` must not emit runes outside rows `floor(bl)..floor(bh)` — adjust the test's row expectations only if the actual rune layout differs from the row derivation in the comment AND the view shows the body confined to the body span; never weaken the "center has wick, sides don't" property.

- [ ] **Step 5: Package suite + commit**

Run: `go test ./canvas/... -count=1 && go vet ./canvas/... && gofmt -l canvas`
Expected: PASS, clean, empty.

```bash
git add canvas/graph/graph.go canvas/graph/candlestick_wide_test.go
git commit -m "feat(graph): DrawCandlestickBottomToTopWide - multi-column candle bodies"
```

---

### Task 2: `DrawCandleWidth` in timeserieslinechart

**Files:**
- Modify: `/Users/evan/projects/ntcharts/linechart/timeserieslinechart/timeserieslinechart.go:348-408` (the `DrawCandle` method)
- Test: Create `/Users/evan/projects/ntcharts/linechart/timeserieslinechart/candle_width_test.go`

**Interfaces:**
- Consumes: Task 1's `graph.DrawCandlestickBottomToTopWide(m, p, width, l, bl, bh, h, s)`.
- Produces: `(m *Model) DrawCandleWidth(openName, highName, lowName, closeName string, bullStyle, bearStyle lipgloss.Style, width int)`; `DrawCandle(...)` unchanged signature, delegates with width 1. Task 3 calls `DrawCandleWidth`.

- [ ] **Step 1: Write the failing test**

Create `/Users/evan/projects/ntcharts/linechart/timeserieslinechart/candle_width_test.go`:

```go
// ntcharts - Copyright (c) 2026 Neomantra Corp.

package timeserieslinechart

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func pushCandles(m *Model) {
	base := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	data := []struct{ o, h, l, c float64 }{
		{100, 108, 97, 105}, {105, 112, 103, 110}, {110, 111, 98, 99},
	}
	for i, d := range data {
		ts := base.AddDate(0, 0, i)
		m.PushDataSet("open", TimePoint{Time: ts, Value: d.o})
		m.PushDataSet("high", TimePoint{Time: ts, Value: d.h})
		m.PushDataSet("low", TimePoint{Time: ts, Value: d.l})
		m.PushDataSet("close", TimePoint{Time: ts, Value: d.c})
	}
}

func newCandleModel(t *testing.T) Model {
	t.Helper()
	m := New(40, 12,
		WithTimeRange(
			time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC)),
		WithYRange(95, 115))
	pushCandles(&m)
	return m
}

// DrawCandle output must be byte-identical to DrawCandleWidth at width 1.
func TestDrawCandleDelegatesAtWidthOne(t *testing.T) {
	s := lipgloss.NewStyle()
	a := newCandleModel(t)
	a.DrawCandle("open", "high", "low", "close", s, s)
	b := newCandleModel(t)
	b.DrawCandleWidth("open", "high", "low", "close", s, s, 1)
	if a.View() != b.View() {
		t.Fatal("DrawCandle and DrawCandleWidth(1) differ")
	}
}

// Width 3 must produce strictly more candle-body columns than width 1.
func TestDrawCandleWidthWidensBodies(t *testing.T) {
	s := lipgloss.NewStyle()
	countBodyCols := func(view string) int {
		cols := 0
		for _, line := range strings.Split(view, "\n") {
			cols += strings.Count(line, "┃")
		}
		return cols
	}
	a := newCandleModel(t)
	a.DrawCandleWidth("open", "high", "low", "close", s, s, 1)
	b := newCandleModel(t)
	b.DrawCandleWidth("open", "high", "low", "close", s, s, 3)
	if countBodyCols(b.View()) <= countBodyCols(a.View()) {
		t.Fatalf("width 3 (%d heavy runes) not wider than width 1 (%d)",
			countBodyCols(b.View()), countBodyCols(a.View()))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./linechart/timeserieslinechart/ -run TestDrawCandle -count=1`
Expected: FAIL to build — `undefined` `DrawCandleWidth` (method does not exist yet).

- [ ] **Step 3: Implement `DrawCandleWidth`, delegate `DrawCandle`**

In `timeserieslinechart.go`, rename the existing `DrawCandle` body into the new method and add the width parameter — the ONLY behavioral change inside the loop is the final draw call switching to the wide primitive:

```go
// DrawCandleWidth is DrawCandle with a candle body width in columns: the
// body spans `width` columns centered on the candle's time-scaled column,
// the wick stays in the center column. Width < 1 is treated as 1.
func (m *Model) DrawCandleWidth(openName, highName, lowName, closeName string, bullStyle, bearStyle lipgloss.Style, width int) {
	// ... existing DrawCandle body verbatim, except the draw call becomes:
	//	graph.DrawCandlestickBottomToTopWide(&m.Canvas,
	//		canvas.Point{X: drawX, Y: m.Origin().Y - 1},
	//		width, lData[i].Y, bl, bh, hData[i].Y, s)
}

// DrawCandle draws single-column candles; kept as the width-1 case of
// DrawCandleWidth for existing callers.
func (m *Model) DrawCandle(openName, highName, lowName, closeName string, bullStyle, bearStyle lipgloss.Style) {
	m.DrawCandleWidth(openName, highName, lowName, closeName, bullStyle, bearStyle, 1)
}
```

Move the doc comment describing the four-dataset contract onto `DrawCandleWidth`; keep the guards (`len(name) == 0`, dataset existence, limit = shortest dataset, `oData[i].X < 0` skip, bull/bear selection) exactly as they are.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./linechart/timeserieslinechart/ -run TestDrawCandle -count=1 -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Package suite + commit**

Run: `go test ./linechart/... -count=1 && go vet ./linechart/... && gofmt -l linechart`
Expected: PASS, clean, empty.

```bash
git add linechart/timeserieslinechart/timeserieslinechart.go linechart/timeserieslinechart/candle_width_test.go
git commit -m "feat(timeserieslinechart): DrawCandleWidth - wide candle bodies"
```

---

### Task 3: rebase `spec.buildOHLC` + auto width + docs

**Files:**
- Modify: `/Users/evan/projects/ntcharts/spec/build.go:497-578` (the `buildOHLC` function and its doc comment)
- Modify: `/Users/evan/projects/ntcharts/spec/helpers.go:10-29` (`pointTime` — add date-only layout)
- Modify: `/Users/evan/projects/ntcharts/spec/build_ohlc_test.go` (update for new model type + new behavior)
- Modify: `/Users/evan/projects/ntcharts/spec/README.md:50-56,71` (drop the clamp/most-recent doc; describe time-scaled placement + auto width)

**Interfaces:**
- Consumes: Task 2's `DrawCandleWidth`; existing `pointTime`, `candleStyle`, `makeTimeAxisFormatter`, `Format.labelFormatter`, `timeserieslinechart.New/WithTimeRange/WithYRange/WithYLabelFormatter/WithXLabelFormatter/PushDataSet/GraphWidth`.
- Produces: `Build` returns `*timeserieslinechart.Model` for `ChartTypeOHLC` (was `*canvas.Model`); package-private `autoCandleWidth(graphWidth int, times []time.Time, tMin, tMax time.Time) int`.

- [ ] **Step 1: Extend `pointTime` with the date-only layout (with test)**

In `spec/helpers.go`, inside the `case string:` block, after the RFC3339 attempt add:

```go
		if t, err := time.Parse("2006-01-02", v); err == nil {
			return t, true
		}
```

Append to `spec/build_ohlc_test.go`:

```go
func TestPointTimeDateOnly(t *testing.T) {
	tm, ok := pointTime("2026-01-05")
	if !ok || !tm.Equal(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("pointTime(date-only) = %v, %v", tm, ok)
	}
}
```

(Add `"time"` to that test file's imports.) Run: `go test ./spec/ -run TestPointTimeDateOnly -count=1` — RED first (fails before the helpers.go edit: `ok == false`), then GREEN after.

- [ ] **Step 2: Write the failing rebase tests**

Replace `TestBuildOHLC` and `TestBuildOHLCClampsToMostRecent` in `spec/build_ohlc_test.go` (keep `ohlcSpec` and `TestBuildOHLCInvertedPinErrors` as they are; swap the `canvas` import for `github.com/NimbleMarkets/ntcharts/v2/linechart/timeserieslinechart`, and add `regexp` and `time` to the imports):

```go
// stripAnsi removes SGR escape sequences so rune positions equal columns —
// candle styles emit color codes that would otherwise inflate byte indices
// and break substring adjacency checks.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiRe.ReplaceAllString(s, "") }

func TestBuildOHLC(t *testing.T) {
	s := ohlcSpec()
	s.Width = 60
	s.XAxis.Format = Format{Kind: "time", Layout: "Jan 02"}
	got, err := Build(s)
	if err != nil {
		t.Fatalf("Build(ohlc): %v", err)
	}
	m, ok := got.(*timeserieslinechart.Model)
	if !ok {
		t.Fatalf("Build(ohlc) returned %T, want *timeserieslinechart.Model", got)
	}
	view := stripAnsi(m.View())
	if !strings.ContainsAny(view, "│┃╽╿") {
		t.Fatalf("no candle runes found in view:\n%s", view)
	}
	// Axis labels render now (the old canvas path drew a bare axis).
	if !strings.Contains(view, "Jan") {
		t.Fatalf("no x-axis time label in view:\n%s", view)
	}
	// Time-scaled placement: candle bodies must reach the right half of
	// the chart, not pack against the left edge. Count in runes (ANSI is
	// stripped above) so index == column.
	rightmost := -1
	for _, line := range strings.Split(view, "\n") {
		runes := []rune(line)
		for i := len(runes) - 1; i >= 0; i-- {
			if runes[i] == '┃' || runes[i] == '╽' || runes[i] == '╿' {
				if i > rightmost {
					rightmost = i
				}
				break
			}
		}
	}
	if rightmost <= s.Width/2 {
		t.Fatalf("all candle bodies in the left half (rightmost col %d) — packed-left bug:\n%s", rightmost, view)
	}
}

func TestBuildOHLCDenseDataStillBuilds(t *testing.T) {
	s := ohlcSpec()
	s.Width = 12
	pts := s.Data.Series[0].OHLC
	for i := 0; i < 40; i++ {
		pts = append(pts, OHLCPoint{
			T: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Format("2006-01-02"),
			O: 100, H: 101, L: 99, C: 100})
	}
	s.Data.Series[0].OHLC = pts
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(ohlc, dense): %v", err)
	}
}

func TestBuildOHLCWideBodies(t *testing.T) {
	s := ohlcSpec()
	s.Width = 60 // 4 candles on a wide chart → auto width > 1
	got, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	view := stripAnsi(got.(*timeserieslinechart.Model).View())
	// A widened body draws the same heavy rune in horizontally adjacent
	// cells; single-column candles never do. (ANSI stripped, or the escape
	// codes between styled cells would defeat the adjacency check.)
	if !strings.Contains(view, "┃┃") {
		t.Fatalf("expected multi-column candle bodies (adjacent heavy runes):\n%s", view)
	}
}

func TestBuildOHLCUnparsableTimeErrors(t *testing.T) {
	s := ohlcSpec()
	s.Data.Series[0].OHLC[0].T = "not-a-time"
	if _, err := Build(s); err == nil {
		t.Fatal("expected unparsable-time error")
	}
}
```

Run: `go test ./spec/ -run TestBuildOHLC -count=1`
Expected: FAIL — `TestBuildOHLC` type assertion (`*canvas.Model, want *timeserieslinechart.Model`); the others fail correspondingly. This is the RED state.

- [ ] **Step 3: Rewrite `buildOHLC`**

Replace the whole function and its doc comment in `spec/build.go` (keep `candleStyle` and the color constants as they are):

```go
// buildOHLC constructs a *timeserieslinechart.Model that renders
// Series.OHLC candlesticks at time-scaled X positions across the full
// graph width, with labeled axes (the same axis machinery as
// ChartTypeTimeSeries). The Y range derives from the data's low/high and
// may be pinned by YAxis.Min/Max (one-sided pins allowed; it is an error
// for the resulting min to exceed the max). Candle body width is chosen
// automatically from chart density (see autoCandleWidth); dense data
// overdraws like any dense timeseries — no points are dropped.
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

	times := make([]time.Time, len(pts))
	for i, p := range pts {
		t, ok := pointTime(p.T)
		if !ok {
			return nil, fmt.Errorf("spec: ohlc point %d has unparsable time %v", i, p.T)
		}
		times[i] = t
	}
	tMin, tMax := times[0], times[0]
	for _, t := range times[1:] {
		if t.Before(tMin) {
			tMin = t
		}
		if t.After(tMax) {
			tMax = t
		}
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

	opts := []timeserieslinechart.Option{
		timeserieslinechart.WithTimeRange(tMin, tMax),
		timeserieslinechart.WithYRange(minP, maxP),
	}
	if yf := s.YAxis.Format.labelFormatter(); yf != nil {
		opts = append(opts, timeserieslinechart.WithYLabelFormatter(yf))
	}
	if !s.XAxis.Format.IsZero() && s.XAxis.Format.Kind == "time" {
		layout := s.XAxis.Format.Layout
		if layout == "" {
			layout = "2006-01-02"
		}
		opts = append(opts, timeserieslinechart.WithXLabelFormatter(
			makeTimeAxisFormatter(layout)))
	}

	m := timeserieslinechart.New(s.Width, s.Height, opts...)
	for i, p := range pts {
		ts := times[i]
		m.PushDataSet("open", timeserieslinechart.TimePoint{Time: ts, Value: p.O})
		m.PushDataSet("high", timeserieslinechart.TimePoint{Time: ts, Value: p.H})
		m.PushDataSet("low", timeserieslinechart.TimePoint{Time: ts, Value: p.L})
		m.PushDataSet("close", timeserieslinechart.TimePoint{Time: ts, Value: p.C})
	}

	up := candleStyle(s.Theme, 0, defaultUpColor)
	down := candleStyle(s.Theme, 1, defaultDownColor)
	m.DrawCandleWidth("open", "high", "low", "close", up, down,
		autoCandleWidth(m.GraphWidth(), times, tMin, tMax))
	return &m, nil
}

// autoCandleWidth picks a candle body width (in columns, odd, >= 1) from
// chart density: roughly an equal share of the graph width per candle,
// clamped so adjacent time-scaled candles cannot overlap on irregular
// calendars, and capped at 7 so sparse charts stay recognizable candles.
func autoCandleWidth(graphWidth int, times []time.Time, tMin, tMax time.Time) int {
	n := len(times)
	if n == 0 || graphWidth < 1 {
		return 1
	}
	w := graphWidth/n - 1
	if n > 1 && tMax.After(tMin) {
		span := float64(tMax.Sub(tMin))
		cols := make([]int, n)
		for i, t := range times {
			cols[i] = int(math.Round(float64(t.Sub(tMin)) / span * float64(graphWidth-1)))
		}
		sort.Ints(cols)
		minGap := graphWidth
		for i := 1; i < n; i++ {
			if g := cols[i] - cols[i-1]; g > 0 && g < minGap {
				minGap = g
			}
		}
		if w > minGap-1 {
			w = minGap - 1
		}
	}
	if w > 7 {
		w = 7
	}
	if w%2 == 0 {
		w--
	}
	if w < 1 {
		w = 1
	}
	return w
}
```

Imports: `spec/build.go` must now import `sort` and `time` if not already present; the `canvas` and `canvas/graph` imports may become unused by this function — remove them ONLY if nothing else in the file uses them (check with `go build ./spec/`).

- [ ] **Step 4: Run the rebase tests to green**

Run: `go test ./spec/ -run 'TestBuildOHLC|TestPointTime' -count=1 -v`
Expected: PASS (5 tests). If `TestBuildOHLCWideBodies` finds no `┃┃`: dump the view and check `autoCandleWidth` — 4 candles on graphWidth ≈ 53 gives share 12, minGap ≈ 17 → width 7 (capped): bodies must span 7 columns, so adjacent heavy runes must exist whenever a candle's body rows render. Debug the width path rather than weakening the assertion.

- [ ] **Step 5: Update spec README**

In `/Users/evan/projects/ntcharts/spec/README.md`, replace the OHLC clamp wording (lines ~50-56) with: candles render at time-scaled X positions across the full graph width with labeled axes; body width is chosen automatically from density (odd, capped at 7, never overlapping); dense series overdraw rather than dropping points. Update the row in the behavior table (line ~71) accordingly (`N/A` column unchanged).

- [ ] **Step 6: Full ntcharts suite + cross-repo verification**

Run: `go test ./... -count=1 && go vet ./... && gofmt -l canvas linechart spec`
Expected: all packages PASS, vet clean, no gofmt output. If any pre-existing test asserts the old packed-left OHLC output, update it for the intended behavior change and justify in the report; any other failure is a regression.

Then: `cd /Users/evan/projects/flint-ntcharts && go test ./... -count=1`
Expected: all 6 packages PASS (speccheck renders the candlestick goldens through the new path).

- [ ] **Step 7: Commit**

```bash
cd /Users/evan/projects/ntcharts
git add spec/build.go spec/helpers.go spec/build_ohlc_test.go spec/README.md
git commit -m "feat(spec): OHLC via timeserieslinechart - time-scaled, labeled, auto-width candles"
git status --short   # only examples/combo_chart.html may remain, unstaged
```
