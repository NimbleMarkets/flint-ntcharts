# flint-ntcharts

A spike proving that [flint-chart](https://github.com/NimbleMarkets/flint-chart) can compile
Vega-Lite specs inside a [Javy](https://github.com/bytecodealliance/javy)/QuickJS WebAssembly
module, run from Go via [wazero](https://github.com/tetratelabs/wazero) — no Node.js runtime
required at execution time. This is Phase 1 (feasibility) of the flint → ntcharts backend
project: proving the wasm approach is viable before committing to it as the production
compile path. See [SPIKE-RESULTS.md](./SPIKE-RESULTS.md) for the full go/no-go verdict,
benchmark numbers, and known caveats.

## Quickstart

The compiled `flint.wasm` module is committed to the repo, so a fresh clone can run the Go
test suite (parity tests + benchmark) with no build step:

```sh
go test ./...
```

### Rebuilding the wasm module

If you change `js/src/*.js` (or want to reproduce the artifact yourself):

```sh
cd js && npm install
make wasm   # downloads bin/javy via `gh` on first run (macOS arm64)
```

### Regenerating reference fixtures

The Node-side reference outputs used by the Go parity test live under `testdata/expected/`.
To regenerate them from `flint-chart` directly:

```sh
cd js && npm run gen-expected
```

## Requirements

- Go >= 1.25 (required by `wazero`)
- Node >= 20
- `gh` CLI on your `PATH` if you need `make wasm` to fetch `bin/javy` (macOS arm64 only; other
  platforms should set `JAVY` to a pre-installed binary)

## Phase 3: TypeScript → ntcharts-spec backend

Phase 3 adds a second, pure-TypeScript compile path that runs `flint-chart`'s core assembly
pipeline (channel semantics, zero-decision, layout, overflow) and instantiates the result
directly into [ntcharts](https://github.com/NimbleMarkets/ntcharts)'
[`spec.Spec`](https://github.com/NimbleMarkets/ntcharts/blob/spec/spec/spec.go) JSON shape,
instead of a Vega-Lite spec. `assembleNtcharts` (`js/src/ntcharts/assemble.ts`) is the entry
point: given a flint `ChartAssemblyInput`, it runs the shared flint-chart layout phases, then
hands off to a per-chart-type template (`js/src/ntcharts/templates/`) that fills in the
`NtSpec` fields (`type`, `data.series`, `x_axis`, `y_axis`, `heat`, `options`, `theme`).

### Supported chart types

- Bar Chart
- Stacked Bar Chart
- Line Chart
- Scatter Plot
- Heatmap

Only **"Line Chart"** is a registered `chartType` name in the template registry
(`js/src/ntcharts/templates/index.ts`) — there is no separate "Timeseries Line Chart" entry.
`"timeseries"` is instead an **emitted** `NtSpec.type` value: the Line Chart template
(`js/src/ntcharts/templates/line.ts`) inspects the resolved X-channel semantics and sets
`emit.type = "timeseries"` when X is temporal, `"line"` otherwise. Feed it a quantitative
(non-temporal) X field and the same `"Line Chart"` chartType emits `type: "line"` — see the
`line-numeric` fixture/golden under `testdata/fixtures-terminal/` and
`testdata/ntspec-golden-terminal/`.

**Grouped Bar Chart is intentionally unsupported.** ntcharts' terminal `barchart.Model` can only
render stacked segments within a bar, not side-by-side groups, so a multi-series `Spec` with
`options.stacked` unset is rejected by `spec.Build` rather than silently stacked. flint-chart's
grouped-bar encodings have no terminal-renderable equivalent in this backend.

### Cells sizing convention

All sizes in the assembler are interpreted in **cells** (terminal character columns/rows), not
pixels. `resolveBaseSizeShim` (`js/src/ntcharts/shims.ts`) clamps a base size (default 64x20) to
an optional canvas-size ceiling from the chart spec; `deriveStretchCapsShim` derives
`maxStretchX`/`maxStretchY` from the same ceiling so flint-chart's shared layout math stretches
plot regions in cell units. The resulting `layout.subplotWidth` / `subplotHeight` become the
emitted `NtSpec.width` / `height` (each floored at a minimum of 8x4 cells).

**Default stretch cap is 1 (no growth beyond the base size).** `TERMINAL_OPTIONS.maxStretch`
(`js/src/ntcharts/assemble.ts`) is `1`: when a chart spec sets no `canvasSize` ceiling,
`deriveStretchCapsShim`'s no-ceiling fallback uses this value, so the emitted chart never
exceeds the requested/default `baseSize` (terminal cells are a fixed budget, not a freely
growable canvas). To explicitly allow growth beyond `baseSize`, set `chart_spec.canvasSize`
larger than `baseSize` (or larger than the 64x20 default when `baseSize` is omitted) — the
per-dimension caps `maxStretchX`/`maxStretchY` are then derived from `canvasSize / baseSize`
instead of the `maxStretch` fallback. See `testdata/fixtures-terminal/` /
`testdata/ntspec-golden-terminal/` for goldens that exercise the default (no-ceiling, capped at
the 64x20 base) path.

### Format translation (d3 → Go)

flint-chart's `FormatSpec` (d3-format style) is translated once per axis
(`js/src/ntcharts/format.ts`) into the ntcharts-spec `Format` directive:

- A `prefix` (e.g. `"$"`) becomes `{ kind: "currency", currency, precision }`.
- A `%`-containing pattern becomes `{ kind: "percent", precision }`.
- `abbreviate` becomes `{ kind: "si", precision: 1 }`.
- Any other pattern becomes `{ kind: "number", precision }` (precision 0 for integer patterns
  like `"d"`).
- d3 format **suffixes** have no ntcharts-spec equivalent and are **dropped**, emitting an
  `info`-severity `format-suffix-dropped` warning.

d3 time-format tokens (`%Y`, `%b`, `%d`, ...) are best-effort translated to Go's reference-time
layout tokens (`2006`, `Jan`, `02`, ...) by `d3TimeToGoLayout`; any untranslatable `%`-token is
passed through verbatim with an `info`-severity `time-format-partial` warning.

### Running the TypeScript test suite

```sh
cd js && npx vitest run
```

### Cross-validating against the real Go renderer

`speccheck/speccheck_test.go` proves the *other* end of the pipeline: every golden JSON file
emitted by the TS backend — both `testdata/ntspec-golden/*.json` (pixel-scale fixtures, an
explicit `baseSize`) and `testdata/ntspec-golden-terminal/*.json` (terminal-scale fixtures, no
`baseSize`, so `DEFAULT_TERMINAL_BASE` 64x20 applies) — unmarshals into ntcharts' `spec.Spec`,
passes `Validate()`, `Build()`s into a concrete terminal model, and renders a non-empty
`View()`. This is the full flint → TS backend → JSON → Go render contract check, run from Go:

```sh
go test ./speccheck/...
```

It requires a local checkout of the `spec` branch of
[ntcharts](https://github.com/NimbleMarkets/ntcharts) at `../ntcharts` relative to this repo
(wired via a `replace` directive in `go.mod`).

### Regenerating the ntcharts-spec goldens

The goldens live under `testdata/ntspec-golden/` (pixel-scale, from `testdata/fixtures/`) and
`testdata/ntspec-golden-terminal/` (terminal-scale, from `testdata/fixtures-terminal/`), and are
produced by `assembleNtcharts` in the vitest suite. To regenerate them after a template or
format change:

```sh
cd js && UPDATE_GOLDEN=1 npx vitest run
```

Goldens additionally carry private `_warnings` / `_width` / `_height` keys (assembly metadata,
not part of the ntcharts-spec schema); the Go side ignores unknown JSON fields, so these are
left in place rather than stripped.

### Known gaps

- **`chart_spec.chartProperties` passes through unvalidated.** Unlike upstream flint-chart,
  this backend does not shim `normalizeChartProperties`: property-driven overrides that
  upstream derives from normalized chartProperties (e.g. `includeZero_x`, `logScale_x`) are
  **not applied** here. See the "Deviations" note at the top of `js/src/ntcharts/shims.ts` for
  the full list of intentionally-unshimmed/unimplemented upstream utilities
  (`normalizeChartProperties`, `computeMinSubplotDimensions`, `decideColorMaps`).
