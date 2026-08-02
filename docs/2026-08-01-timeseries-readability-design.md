# Timeseries Readability Design

**Date:** 2026-08-01
**Status:** Approved
**Scope:** Two fixes — fitted y-domain pinning in the TS backend (this repo)
and final-x-label right-alignment in ntcharts' linechart renderer
(`../ntcharts`, `spec` branch).

## Problem

A timeseries line chart of prices in the 101–121 range renders squashed into
the top three rows of the terminal chart, above a y-axis that starts at
$0.00, and the right end of the x axis — under the newest data — has no time
label.

Root causes (verified by probe renders):

1. **Y squash.** The line/scatter templates implement only half of flint's
   zero-decision: `if (cs.y?.zero?.zero) emit.y_axis.min = 0` — and emit
   nothing when zero is *excluded*. ntcharts' `buildTimeSeries`/`buildLine`/
   `buildScatter` only call `WithYRange` when the spec pins a bound, and the
   underlying `timeserieslinechart.New` starts its view at 0 with an
   auto-range that only expands. Net: an unpinned spec always gets a $0
   baseline. Upstream flint's ECharts backend handles the negative case
   (`instantiate-spec.ts:354`): `!zero && domainPadFraction > 0` →
   `computePaddedDomain(values, padFraction)` → pin `min`/`max`. The ntcharts
   backend never ported that branch.
2. **Missing final x label.** `linechart.go`'s `drawXLabel` walks tick
   positions left to right and refuses any label whose text would extend past
   the canvas (`len(s) + origin.X + i <= canvas.Width()`). At the final tick
   (`i == last`, the axis end) that check nearly always fails, so the newest
   data is never labeled.

Candlestick is unaffected (ntcharts `buildOHLC` derives its y-range from
low/high data). Bars keep `min = 0` by design (zero-decision forces zero for
length-encoded marks). Heatmap/sparkline have no continuous y axis to pin.

## Part 1 — fitted y-domain (this repo)

New `js/src/ntcharts/domain.ts` exporting one helper:

```ts
// Pin emit.y_axis.min/max to a padded data-fitted domain when flint's
// zero-decision excludes zero — the terminal renderer otherwise defaults to
// a 0 baseline. Mirrors the upstream ECharts backend's fitted-domain branch.
export function pinFittedYDomain(emit, cs, table): void
```

Behavior, mirroring upstream: no-op unless `cs.y?.zero` exists,
`cs.y.zero.zero === false`, and `cs.y.zero.domainPadFraction > 0`. Collect
`table` rows' `cs.y.field` values that are finite numbers, call flint-chart's
exported `computePaddedDomain(values, cs.y.zero.domainPadFraction)`, and on a
non-null result set `emit.y_axis.min`/`emit.y_axis.max` (creating `y_axis` is
not needed — both callers already populated it). No y2 pairing (no range-band
charts in this backend).

Call sites: `line.ts` and `scatter.ts`, as the `else` of their existing
`if (cs.y?.zero?.zero) emit.y_axis.min = 0;` line.

Ripple (the standard verification train):

- Regenerate ntspec goldens (`UPDATE_GOLDEN=1 npx vitest run`) — expected
  diffs only in `line-temporal`, `line-numeric`, `scatter` (both pixel and
  terminal scales, where the fixture exists).
- Regenerate Node reference fixtures (`npm run gen-expected`).
- Rebuild `compile/flint.wasm` + `flint.wasm.buildinfo` (`make wasm`).
- `go test ./...` re-proves parity (13/13 fixtures — count unchanged, no
  drift-guard bump), speccheck, and the flint-edit examples suite.
- A vitest assertion pins the property itself: the line-temporal golden's
  `y_axis.min` is > 0 and < the data minimum, `y_axis.max` > the data
  maximum (i.e. fitted and padded, not zero-based).

No new warnings are emitted, so flint-edit's `TestExamplesCompile`
(state 1 = warning-free) stays green.

## Part 2 — final x label (../ntcharts, `spec` branch)

In `linechart/linechart.go` `drawXLabel`, only the final step changes: at
`i == last`, when the label does not fit left-anchored, right-align it —
draw at `X = canvas.Width() - len(s)` — subject to the same guards as every
other label: the cell left of the draw position must be empty (no collision
with an earlier label) and the value must differ from the previously drawn
label. Mid-axis ticks are untouched.

Test (ntcharts repo): a spec-level Build+View test renders a timeseries at a
width where the final month label previously vanished and asserts it now
appears at the right edge; the existing linechart/spec suites guard the rest.
flint-ntcharts' speccheck asserts non-empty views only, so it is insensitive
to this change.

## Verification (end-to-end)

With both parts merged, render the flint-edit Timeseries Line example via
`tui.Render` at 45x20: the y axis starts near the padded data minimum
(~$96, not $0.00) and the final time label is present at the right edge.
Part 1's golden property test and Part 2's ntcharts view test each pin their
half permanently.

## Ordering

Part 1 lands first (feature branch in this repo, SDD flow), then Part 2 as a
commit on ntcharts' `spec` branch. Independent — neither breaks the other's
suite in either order.
