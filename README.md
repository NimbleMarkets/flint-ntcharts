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
(`compile/flint.wasm.buildinfo`) are committed to the repo, so no *build* step (no Javy, no
`js/` install) is needed to run the Go test suite. A fresh clone still needs one thing before
`go test ./...` will pass, though: both `compile/` and `speccheck/` decode into
[`ntcharts`](https://github.com/NimbleMarkets/ntcharts)' `spec.Spec` type directly, so
`go.mod`'s `replace github.com/NimbleMarkets/ntcharts/v2 => ../ntcharts` directive requires a
sibling checkout of `ntcharts` (`spec` branch) at `../ntcharts` relative to this repo — see
"Cross-validating against the real Go renderer" below for how to get one. With that sibling in
place:

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
- `ctx` is honored for cancellation: `New` builds the wazero runtime with
  `WithCloseOnContextDone(true)`, so an already-canceled (or later-canceled) `ctx` aborts the
  compile and `Compile`/`CompileRaw` return a non-nil error instead of a result.

#### Envelope contract

`Compile`/`CompileRaw` parse the same JSON envelope the wasm module (and the Node reference
runner used by `npm run gen-expected`) emit on stdout — frozen as of Phase 4:

```json
{"spec": <NtSpec without any _-prefixed keys>, "warnings": [ChartWarning, ...], "size": {"width": N, "height": N}}
```

`warnings` is always present (possibly `[]`); `size` is always present; key order is exactly
`spec, warnings, size`; the wasm/Node path serializes it with `JSON.stringify` and no
pretty-printing. `spec` is valid input to ntcharts' `spec.Build`.

There are two failure layers, both specified:

1. **Compile-time failure** (e.g. an unsupported chart type). The wasm/Node compiler catches
   this internally, still writes valid JSON to stdout, and exits normally — success vs failure
   is discriminated by the top-level key present, not by process exit status or a thrown
   exception:

   ```json
   {"error": {"message": "Unknown chart type \"Rose Chart\". Supported: ..."}}
   ```

   `CompileRaw` returns these bytes verbatim (no error) — callers that call `CompileRaw`
   directly must check for the `error` key themselves. `Compile` parses the envelope for you
   and, when it finds an `error` key instead of `spec`/`warnings`/`size`, returns a non-nil Go
   `error` built from `error.message` rather than a zero-value `spec.Spec`.
2. **Wasm TRAP** (an engine-level failure, e.g. OOM, or a canceled `ctx` — see above). This
   never produces the JSON envelope at all; it surfaces as a process-level Go `error` from
   `CompileRaw` (and therefore `Compile`) with the module's stderr attached to the error
   message.

### Build provenance

`compile/flint.wasm.buildinfo`, written by `make wasm`, records the Javy version, the
`flint-chart` npm package version, the sha256/byte size of the pre-Javy esbuild bundle, and
the sha256 of the resulting `compile/flint.wasm` itself — enough to check whether the
committed wasm matches a given `js/` checkout without rebuilding it:

```
javy: javy 9.0.0
flint-chart: 0.2.1
bundle-sha256: 783631d9be839b99ca816ec3aeb06c284240a96f3e53f982bc9af52f979a9cc2
bundle-bytes: 675508
wasm-sha256: 537c886e481ebbd852e9836b2089a5ea2eb60ce07dfba7c4add797cda1fc5cc8
```

`bundle-sha256` is also re-derived and checked in CI (see "Continuous Integration" below): the
`Wasm provenance check` step rebuilds the esbuild bundle (but not the wasm module itself, no
Javy involved) and asserts its hash matches this file's committed `bundle-sha256`, binding the
committed `flint.wasm` to the committed `js/src/` that's supposed to have produced it.

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

## flint-tui

`cmd/flint-tui` builds a persistent terminal chart window: a `bubbletea`/`lipgloss`
program (`tui.Model`, `tui.New(runner)`) that stays open, watching one or more sources
for documents to compile and render, re-fitting the chart to the terminal on every
resize. This is the intended integration point for agents and scripts that want a live
chart rather than a one-shot render: push a new document at any time and the window
updates in place.

```sh
go build -o flint-tui ./cmd/flint-tui

# watch a file (poll interval defaults to 250ms)
flint-tui chart.json
flint-tui --poll 100ms chart.json

# read a stream of JSON documents from stdin
flint-tui --stdin

# listen on a unix socket for document streams
flint-tui --listen /tmp/flint.sock
```

At least one source is required; passing none prints usage to stderr and exits 2.
Sources may be combined (e.g. `flint-tui --stdin --listen /tmp/flint.sock chart.json`)
— whichever source delivers a document most recently is the one on screen. `--stdin`
and `--listen PATH` both accept a stream of back-to-back or NDJSON-style JSON documents
(`encoding/json.Decoder` semantics — pretty-printed or compact, no delimiter required).

### Document sniffing

Each document pushed to `flint-tui`, from any source, is sniffed by its top-level keys
and compiled/built accordingly:

| Top-level key | Document kind | Handling |
| --- | --- | --- |
| `chart_spec` | flint `ChartAssemblyInput` | run through the embedded wasm compiler (`compile.Runner.Compile`), sized to the current terminal via `compile.WithBaseSize` |
| `spec` | a `compile.Compile`/`CompileRaw` envelope | the `spec` field is lifted out and re-sized to the current terminal |
| `type` | a raw `ntcharts/v2/spec.Spec` document | re-sized to the current terminal and built directly |

Anything else (or invalid JSON) is a sniff error, surfaced on the status line without
discarding whatever chart was on screen already (see below).

### Fit-to-window

The chart is always re-compiled/re-built at `terminal width x (terminal height - 1)`
(one row reserved for the status line) — both on receiving a new document and on every
`tea.WindowSizeMsg` (terminal resize), so an already-displayed chart re-fits without
needing a fresh document push.

### Keys

- `q` or `ctrl+c` — quit

### Status line and error handling

The bottom row always shows `flint-tui · <source> · <width>x<height>`. If the most
recent document failed to sniff, compile, or build, the compiler/validation error is
appended **verbatim** and the previously-rendered chart is left on screen (nothing is
blanked out just because the latest push was bad) — so a typo in an agent's next push
never blanks a working dashboard. Non-fatal compiler warnings are shown the same way,
one at a time, when there is no error.

### Agent workflow example

A typical agent loop compiles a flint document with
[`flint-chart-mcp`](https://github.com/NimbleMarkets/flint-chart), reshapes it with
`jq` if needed, and pushes the result at a live `flint-tui` window over its unix
socket:

```sh
flint-tui --listen /tmp/flint.sock &

flint-chart-mcp compile spec.json | jq '.' | nc -U /tmp/flint.sock
```

Or, for the simplest possible loop (no socket, just a watched file):

```sh
flint-tui watched.json &
# ... later, from anywhere ...
cat chart.json > watched.json
```

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
run build`, step "Bundle builds (wasm source)" — a compile-only sanity check, the resulting
bundle isn't Javy-built into a new wasm module), checks that bundle's sha256 against the
`bundle-sha256` committed in `compile/flint.wasm.buildinfo` ("Wasm provenance check" — see
"Build provenance" above; this is what actually binds the committed `js/src/` to the
committed `flint.wasm` without needing Javy in CI), and finally runs `go build ./...`
(this now also compiles `cmd/flint-tui`) followed by `go test ./...`, both gated on the
`ntcharts` sibling checkout below.

**Known gap — the `ntcharts` sibling checkout.** Both `go test ./compile/...` (it decodes the
envelope's `spec` field directly into ntcharts' `spec.Spec`) and `go test ./speccheck/...` (it
cross-validates the emitted ntcharts-spec JSON against the real ntcharts Go renderer) need the
`ntcharts` module checked out at `../ntcharts` (wired via the `replace` directive in
`go.mod`), which doesn't exist in CI until the `ntcharts` repo/branch this depends on has a
pushed remote the workflow can check out alongside this one. The workflow probes for
`../ntcharts` in a "Check ntcharts sibling" step and skips the single `go test ./...` step
with a `::warning::` annotation when it's absent, rather than hard-failing the whole run —
there is no longer a Go test package in this module that can run without the sibling. Revisit
this guard once `../ntcharts` is available in CI (checking out the sibling repo explicitly, or
vendoring it). This repo itself has no remote yet either, so the workflow's first real
execution happens on the first push.

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
