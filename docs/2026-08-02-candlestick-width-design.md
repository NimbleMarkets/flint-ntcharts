# Candlestick Width & Time-Scaling Design

**Date:** 2026-08-02
**Status:** Approved
**Scope:** ntcharts only (`../ntcharts`, `spec` branch) — `spec/build.go`,
`linechart/timeserieslinechart/`, `canvas/graph/`. No flint-ntcharts code or
spec-format changes; the emitted `OHLCPoint` already carries `t`.

## Problem

The candlestick chart occupies a tiny fraction of the x axis and has no axis
labels. Root cause (verified by probe renders and source): `spec.buildOHLC`
draws candle *i* at column `origin.X + 1 + i` on a raw canvas — one column
per candle, packed left, ignoring both chart width and the points' time
values, with a bare `DrawXYAxis` and no labels. Separately, no layer of the
stack has a candle-width concept: `graph.DrawCandlestickBottomToTop` is a
single-column primitive, and even `timeserieslinechart.DrawCandle` (the
existing, better path `buildOHLC` fails to use) draws 1-cell candles at
time-scaled positions.

## Part A — rebase `buildOHLC` onto timeserieslinechart

`buildOHLC` constructs a `timeserieslinechart.Model` with the same option
wiring `buildTimeSeries` uses:

- `WithTimeRange(minT, maxT)` from the parsed `OHLCPoint.T` values (reuse the
  spec package's existing time-coercion used for `DataPoint.X`; `T` is
  `time.Time`, RFC3339 string, or ms-since-epoch).
- `WithYRange(minLow, maxHigh)` from the data, overridden by `YAxis.Min`/`Max`
  pins with the same one-sided rule and min>max error as today's `buildOHLC`.
- Y label formatter from `YAxis.Format`; X time-format layout from
  `XAxis.Format` when `kind == "time"` — identical wiring to
  `buildTimeSeries`.

It pushes four datasets (`open`, `high`, `low`, `close`) via `PushDataSet`
(one `TimePoint` per candle per dataset, identical times across the four) and
draws via `DrawCandleWidth` (Part B) with bull/bear styles from the theme
(same colors as today). The returned model's `View()` is ready immediately,
matching the `Build` contract. Behavior changes accepted as intended:
candles spread across the full width at time-scaled positions; x and y axis
labels now render (via `DrawXYAxisAndLabel` inside the candle draw path,
inheriting the final-label fix); the old "more candles than columns → drop
oldest" rule disappears (dense data overdraws like any dense timeseries).

## Part B — candle width by composition

New primitive in `canvas/graph`:

```go
// DrawCandlestickBottomToTopWide draws a candle whose body spans `width`
// columns centered on p.X; the wick renders only in the center column.
// width < 1 is treated as 1; width 1 is exactly DrawCandlestickBottomToTop.
func DrawCandlestickBottomToTopWide(m *canvas.Model, p canvas.Point, width int, l, bl, bh, h float64, s lipgloss.Style)
```

Implementation is composition, not new rune logic: the center column draws
the existing full candle `(l, bl, bh, h)`; each side column draws the same
primitive with `(bl, bl, bh, bh)` — a body-only candle. Even widths render as
the next lower odd width plus one extra column on the right (side columns are
`p.X ± 1 .. p.X ± ⌊(width−1)/2⌋`, plus `p.X + width/2` when width is even);
columns outside the canvas are clipped by the underlying primitive.

In `timeserieslinechart`:

```go
// DrawCandleWidth is DrawCandle with a candle body width in columns.
func (m *Model) DrawCandleWidth(openName, highName, lowName, closeName string, bullStyle, bearStyle lipgloss.Style, width int)
```

`DrawCandle` becomes `DrawCandleWidth(..., 1)` — existing callers see
byte-identical output.

`buildOHLC` auto-computes the width (no spec knob — YAGNI):

```
width = max(1, graphWidth/n − 1)          // n = candle count
width = min(width, minAdjacentGap − 1)    // gap between adjacent time-scaled centers, ≥1
if width % 2 == 0 { width-- }             // odd, so bodies center on wicks
```

`minAdjacentGap` is computed from the actual scaled column positions, so
irregular calendars (weekends, gaps) cannot cause neighboring bodies to
overlap (the clamp is skipped when n == 1, where no gap exists). The auto
width is additionally capped at **7 columns** so sparse charts (one or two
candles) stay recognizable candles rather than bars filling the plot.

## Testing (RED first, ntcharts repo)

1. `canvas/graph`: width-3 candle spans exactly columns `x−1..x+1` for body
   rows, wick runes only at `x`; width-1 output identical to the existing
   primitive on the same inputs.
2. `spec`: Build+View of a ChartTypeOHLC spec at width ~60 with 7 daily
   candles asserts (a) candle runes appear in the right half of the view
   (kills packed-left), (b) an x time label renders, (c) bodies are wider
   than one column.
3. Existing OHLC tests (`build_ohlc_test.go` and any golden views) updated
   for the intended output change, each justified in the report; any other
   test difference is a regression.
4. Cross-repo: full flint-ntcharts suite green against the modified sibling
   (speccheck builds/renders the candlestick goldens through this path).

## Verification (end-to-end)

flint-edit Candlestick example (alt+3) rendered via `tui.Render` at 45x20+:
candles distributed across the full x axis with multi-column bodies and
labeled axes.
