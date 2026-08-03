# Candle Edge Insets & Block Style Design

**Date:** 2026-08-03
**Status:** Approved
**Scope:** ntcharts (`../ntcharts`, `spec` branch): `canvas/graph`,
`linechart/timeserieslinechart`, `spec`. flint-ntcharts: `candlestick.ts`
template passthrough + wasm/provenance rebuild (goldens byte-identical) +
flint-edit example update.

## Problems

1. **Edge candles render half-clipped.** The first candle's center sits at
   graph column 0 and the last is clamped to the final canvas column, so the
   graph-area guard (left) and canvas edge (right) eat their side columns —
   roughly half of each edge body is missing.
2. **Candle bodies look "half-filled".** The renderer's only rune set is
   heavy box-drawing lines (`┃ ╻ ╹ ╽ ╿`), which occupy about half of each
   cell. There is no solid-block alternative and no way to choose one.

Colors are NOT a gap: `Theme.Palette[0]` (bull) / `Palette[1]` (bear)
already drive candle colors from any spec.

## Part 1 — edge insets (ntcharts)

In `DrawCandleWidth`'s per-candle loop, clamp the candle **center** so the
whole body fits inside the drawable span, instead of clipping its side
columns:

```
half    = (width-1)/2                 // left reach
right   = width-1-half                // right reach (covers even widths)
minCtr  = firstGraphCol + half        // firstGraphCol = Origin().X (+1 when YStep>0)
maxCtr  = (Canvas.Width()-1) - right
center  = clamp(drawX, minCtr, maxCtr)
```

When `maxCtr < minCtr` (chart narrower than one candle) fall back to the
current behavior (center clamped to the drawable span, side columns clipped
by the existing guards). The existing left-boundary `continue` and canvas
clipping stay as the safety net for dense data whose interior candles
overlap. Edge candles shift inward up to `half` columns from their true
time position — deliberate plot-inset behavior, documented on
`DrawCandleWidth`. Width 1 reduces to the existing newest-candle clamp
(`half == right == 0`) — no behavior change.

## Part 2 — block style

### Renderer primitive (`canvas/graph`)

New standalone primitive (not a parametrization of the line one):

```go
// DrawCandlestickBlockBottomToTop draws a candle with a solid block body:
// full-body cells are FullBlock, the body's boundary cells use lower/upper
// half blocks by the same half-rounding rule the line style applies
// (┃→█, ╻→▄, ╹→▀), and wick cells use the light line runes (│ ╵ ╷). Where
// a body boundary and a wick share a cell, the block wins and the wick
// continues in the adjacent cell.
func DrawCandlestickBlockBottomToTop(m *canvas.Model, p canvas.Point, l, bl, bh, h float64, s lipgloss.Style)
```

`▀` (U+2580, upper half block) is added to `canvas/runes` if not already
present; `▄` is `LowerBlockFour`, `█` is `FullBlock`.

### timeserieslinechart

```go
type DrawCandleOpts struct {
	Width int  // body width in columns; <1 treated as 1
	Block bool // solid block bodies instead of line runes
}

func (m *Model) DrawCandleWithOpts(openName, highName, lowName, closeName string, bullStyle, bearStyle lipgloss.Style, opts DrawCandleOpts)
```

`DrawCandleWidth(..., w)` delegates to `DrawCandleWithOpts` with
`{Width: w}`; `DrawCandle` remains the `{Width: 1}` case. The per-column
loop selects the line or block primitive per cell-column (center = full
candle, sides = body-only), same geometry either way. Part 1's inset lives
in `DrawCandleWithOpts` so both styles get it.

### ntcharts-spec

`Options.CandleStyle string` (JSON `candle_style`, omitempty): `""` and
`"line"` mean the current line style; `"block"` selects the block
primitive; any other value is a `Build` error
(`spec: unknown candle_style %q`). `buildOHLC` maps it into
`DrawCandleOpts.Block`. README documents the option in the Options table
and the OHLC section.

## flint-ntcharts side

- `js/src/ntcharts/templates/candlestick.ts`: when
  `chart_spec.chartProperties?.candleStyle` is `"line"` or `"block"`, emit
  `options.candle_style` accordingly; other values are ignored (consistent
  with chartProperties passing through unvalidated — no new warning).
- Fixtures don't set the property, so **all goldens and Node reference
  fixtures stay byte-identical**; the change still requires the wasm
  bundle/provenance rebuild (`make wasm`) because the template source
  changed.
- The flint-edit Candlestick example (`cmd/flint-edit/examples.go`) adds
  `"chartProperties": { "candleStyle": "block" }` so the demo renders solid
  candles; `TestExamplesCompile` keeps guarding it compiles warning-free.
- A vitest case asserts the passthrough: candlestick input with
  `chartProperties.candleStyle: "block"` emits
  `options.candle_style == "block"`, and an unrelated value emits nothing.

## Testing (RED first)

ntcharts:
1. Inset: a wide-candle spec render shows the first candle's full body
   (body runes present in the `half` columns right of the axis) and the
   last candle's full body (present in the final `width` columns), with the
   y-axis column intact — extends the existing boundary test.
2. Block primitive: full-body cells are `█`, boundary cells `▄`/`▀` per
   half-rounding, wick cells line runes; side-by-side with the line
   primitive on the same inputs, cell occupancy (null/non-null) is
   identical.
3. Spec: `candle_style: "block"` Build+View contains `█`; `"line"`/empty
   render as today; `"bogus"` errors.

flint-ntcharts: the vitest passthrough test above; goldens/expected
asserted unchanged after regen; full Go train green (parity, speccheck,
flint-edit examples).

## Verification (end-to-end)

flint-edit alt+3 renders solid-block candles, full bodies at both edges,
labeled axes.
