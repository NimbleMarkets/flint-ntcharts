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
- Line Chart / Timeseries Line Chart
- Scatter Plot
- Heatmap

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
emitted by the TS backend (`testdata/ntspec-golden/*.json`) unmarshals into ntcharts'
`spec.Spec`, passes `Validate()`, `Build()`s into a concrete terminal model, and renders a
non-empty `View()`. This is the full flint → TS backend → JSON → Go render contract check, run
from Go:

```sh
go test ./speccheck/...
```

It requires a local checkout of the `spec` branch of
[ntcharts](https://github.com/NimbleMarkets/ntcharts) at `../ntcharts` relative to this repo
(wired via a `replace` directive in `go.mod`).

### Regenerating the ntcharts-spec goldens

The goldens live under `testdata/ntspec-golden/` and are produced by `assembleNtcharts` in the
vitest suite. To regenerate them after a template or format change:

```sh
cd js && UPDATE_GOLDEN=1 npx vitest run
```

Goldens additionally carry private `_warnings` / `_width` / `_height` keys (assembly metadata,
not part of the ntcharts-spec schema); the Go side ignores unknown JSON fields, so these are
left in place rather than stripped.
