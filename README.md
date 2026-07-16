# flint-ntcharts

Compiles [flint-chart](https://github.com/NimbleMarkets/flint-chart) `ChartAssemblyInput`
specs into an [ntcharts](https://github.com/NimbleMarkets/ntcharts) `spec.Spec` envelope —
the flint → ntcharts terminal-UI compile backend. The TypeScript assembly pipeline
(`js/src/ntcharts/`) is bundled into a [Javy](https://github.com/bytecodealliance/javy)/QuickJS
WebAssembly module and run from Go via [wazero](https://github.com/tetratelabs/wazero), so no
Node.js runtime is required at execution time.

This started as a Phase 1 feasibility spike proving the wasm approach viable, using a
stand-in Vega-Lite compile as the test payload before the real ntcharts-spec output existed —
that wording is now historical; see [SPIKE-RESULTS.md](./SPIKE-RESULTS.md), a dated go/no-go
record left untouched from that phase. Phase 3 added the pure-TypeScript ntcharts-spec
backend described below, and Phase 4 wired it into the wasm module and the production Go
API (`compile.New` / `compile.Compile`) documented in the Quickstart.

## Quickstart

The compiled `flint.wasm` module and its build provenance
(`compile/flint.wasm.buildinfo`) are committed to the repo, so a fresh clone runs the full
Go test suite (parity + speccheck, no build step) with:

```sh
go test ./...
```

### Go API

```go
import (
    "context"

    "github.com/NimbleMarkets/flint-ntcharts/compile"
)

r, err := compile.New(ctx)
if err != nil {
    // handle
}
defer r.Close(ctx)

spec, warnings, err := r.Compile(ctx, chartAssemblyInputJSON, compile.WithBaseSize(80, 24))
if err != nil {
    // handle
}
// spec is a github.com/NimbleMarkets/ntcharts/v2/spec.Spec, ready for spec.Build(spec).
```

- `compile.New(ctx)` compiles the embedded `flint.wasm` once and returns a `*Runner`, safe
  for concurrent `Compile`/`CompileRaw` calls — each call instantiates a fresh, isolated
  module instance. `NewRunner` is a deprecated alias kept for source compatibility.
- `(*Runner).Compile` returns `(spec.Spec, []Warning, error)`; `warnings` is never nil.
  `(*Runner).CompileRaw` returns the raw envelope JSON bytes instead, for callers that want
  to parse it themselves.
- `compile.WithBaseSize(w, h int)` sets `chart_spec.baseSize` (terminal cells) — the
  requested render size. The backend runs with a stretch cap of 1 (see "Default stretch cap"
  below), so this is a hard bound unless paired with `compile.WithCanvasSize(w, h int)`,
  which sets a growth ceiling above `baseSize`.

#### Envelope contract

`Compile`/`CompileRaw` parse the same JSON envelope the wasm module (and the Node reference
runner used by `npm run gen-expected`) emit on stdout — frozen as of Phase 4:

```json
{"spec": <NtSpec without any _-prefixed keys>, "warnings": [ChartWarning, ...], "size": {"width": N, "height": N}}
```

`warnings` is always present (possibly `[]`); `size` is always present; key order is exactly
`spec, warnings, size`; the wasm/Node path serializes it with `JSON.stringify` and no
pretty-printing. `spec` is valid input to ntcharts' `spec.Build`.

### Build provenance

`compile/flint.wasm.buildinfo`, written by `make wasm`, records the Javy version, the
`flint-chart` npm package version, and the sha256/byte size of the pre-Javy esbuild bundle
that produced the committed `compile/flint.wasm` — enough to check whether the committed
wasm matches a given `js/` checkout without rebuilding it:

```
javy: javy 9.0.0
flint-chart: 0.2.1
bundle-sha256: 16f7eddaad3bc03e4df99fb1d3cb9653b637fe43735bf01c2eaed523fc8d4dbe
bundle-bytes: 675065
```

### Rebuilding the wasm module

If you change `js/src/*.ts`/`js/src/*.js` (or want to reproduce the artifact yourself):

```sh
cd js && npm install
cd ..
make wasm   # downloads bin/javy on first run; picks the right release asset for your OS/arch
```

`make wasm` detects the host OS/architecture via `uname` (macOS/Linux, arm64/x86_64) and
downloads the matching Javy release asset with `gh release download`, then regenerates both
`compile/flint.wasm` and `compile/flint.wasm.buildinfo`. On an unsupported OS/arch, set
`JAVY=/path/to/javy` to point at a pre-installed binary instead.

### Regenerating reference fixtures

The Node-side reference outputs used by the Go parity test live under `testdata/expected/`
(pixel-scale) and `testdata/expected-terminal/` (terminal-scale). To regenerate them from
`flint-chart` directly:

```sh
cd js && npm run gen-expected
```

CI runs this same command and fails the build if it produces a diff (see "Continuous
Integration" below) — regenerate and commit whenever a fixture or backend change should
change the reference output.

## Requirements

- Go >= 1.25 (required by `wazero`)
- Node >= 20
- `gh` CLI on your `PATH` if you need `make wasm` to fetch `bin/javy` (auto-detects
  macOS/Linux, arm64/x86_64 via `uname`; on any other platform, set `JAVY` to a pre-installed
  binary instead)

## Continuous Integration

`.github/workflows/ci.yml` validates the artifacts already committed to the repo on every
push to `main` and every pull request. It does **not** invoke Javy or rebuild `flint.wasm` —
that stays a local/manual `make wasm` step (see "Rebuilding the wasm module" above); running
Javy in CI is a later nicety. In order, the workflow: installs JS deps (`npm ci`),
typechecks and runs the vitest suite (`tsc --noEmit`, `vitest run`), regenerates the
Node-reference fixtures and fails on any diff against the committed
`testdata/expected{,-terminal}/` (reference-drift check), builds the esbuild bundle (`npm
run build`, a compile-only sanity check — the resulting bundle isn't Javy-built or
compared), and runs `go test ./compile/` against the committed `flint.wasm` (byte-parity, no
`ntcharts` dependency needed).

**Known gap — the `ntcharts` sibling checkout.** `go test ./speccheck/...` needs the
`ntcharts` module checked out at `../ntcharts` (wired via the `replace` directive in
`go.mod`), which doesn't exist in CI until the `ntcharts` repo/branch this depends on has a
pushed remote the workflow can check out alongside this one. The workflow probes for
`../ntcharts` in a "Check ntcharts sibling" step and skips `go test ./speccheck/` with a
`::warning::` annotation when it's absent, rather than hard-failing the whole run; `go test
./compile/` (parity) has no `ntcharts` import and always runs regardless. Revisit this guard
once `../ntcharts` is available in CI (checking out the sibling repo explicitly, or vendoring
it). This repo itself has no remote yet either, so the workflow's first real execution
happens on the first push.

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
