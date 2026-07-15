# Flint → ntcharts Backend Design

**Date:** 2026-07-15
**Status:** Approved
**Scope:** Add [ntcharts](https://github.com/NimbleMarkets/ntcharts) as a Flint
rendering backend, with a companion Go TUI hosting app and a
[booba-shim](https://github.com/NimbleMarkets/booba-shim) subpackage for
browser-hosted Go-WASM TUIs.

## Summary

One JSON contract — **ntcharts-spec** — sits at the center. Flint's compiler
(unchanged, Stages 1+2) plus a new out-of-tree Stage-3 backend produce it;
`ntcharts/spec.Build()` consumes it. The Flint compiler reaches native Go as an
embedded WASM module (esbuild → Javy → wazero) and reaches the browser through
a booba-shim wrapping the JS library directly. Same contract on every path.

```
                       TS: @nimblemarkets/flint-ntcharts (Stage-3 backend)
  Flint input ──► flint-chart core (Stage 1+2) ──► assembleNtcharts() ──► ntcharts-spec JSON
                                                        │
                    ┌───────────────────────────────────┼──────────────────────┐
                    │ native Go                         │ browser (go-booba)   │ hand-authored /
                    │ flint.wasm (Javy bundle)          │ booba-shim/flintchart│ config files
                    │ run by wazero, go:embed           │ calls flint JS direct│
                    └───────────────┬───────────────────┴──────────┬───────────┘
                                    ▼                              ▼
                          ntcharts/spec: Build(s) ──► ntcharts models ──► terminal / wasm terminal
                                    └── s.ToECharts() ──► web (unchanged bonus surface)
```

## Decisions and rationale

| Decision | Choice | Rationale |
| --- | --- | --- |
| Compiler distribution | JS→WASM via Javy (QuickJS), executed with wazero | Single TS source of truth, zero drift, self-contained pure-Go binary — no cgo, no Node. Compile latency (ms–tens of ms per chart under QuickJS) is negligible for a TUI. |
| TS backend placement | Out-of-tree npm package `@nimblemarkets/flint-ntcharts` | flint-chart is a Microsoft repo; out-of-tree keeps release cadence under NimbleMarkets control. Upstream later if wanted. `docs/adding-a-backend.md` documents the Stage-3 contract; backends consume exported core APIs. |
| Render target | Evolve the existing experimental `ntcharts/spec` package (spec branch) into ntcharts-spec v1 | Something concrete must be the wire format; an unnamed private JSON contract is still a spec, minus reuse value. ntcharts-spec takes the same structural position Vega-Lite/ECharts specs hold for the other backends. Flint relieves it of the "smart" parts (inference, layout), narrowing it to the resolved/concrete altitude. |
| Repo layout | New repo `NimbleMarkets/flint-ntcharts` for TS backend + wasm pipeline + Go compile pkg + TUI host | Everything speaking the flint↔ntcharts contract versions together. ntcharts stays dependency-lean (no wazero, no flint knowledge). booba-shim subpackage lives in booba-shim per its pattern. |
| TUI host shape | Live chart server | A persistent terminal chart window agents push specs into — the "hosting" story. One-shot viewing falls out of file-watching. |
| Faceting | Out of scope for v1 | Big terminal-layout question; legitimate future work. |

## Components

### 1. ntcharts-spec v1 (in `NimbleMarkets/ntcharts`, graduating the `spec` branch)

Keep the existing philosophy: neutral, JSON-marshalable, every option
optional-with-autoscale, surfaces free to ignore what they can't do.
Hand-authors lean on ntcharts autoscaling; Flint emits fully-pinned specs.
Breaking changes to the experimental format are allowed.

Changes from the current spec branch:

- **Axis structs** replace loose fields: `XAxis{Title, Type, Labels, Format}`,
  `YAxis{Title, Min, Max, Format}` (absorbing `Options.YAxisMin/Max` and
  `Options.TimeFormat`).
- **Format directive** is a small Go-friendly struct —
  `Format{Kind: number|percent|currency|si|time, Precision, Currency, Layout}`
  — **not** a d3 format string. The TS backend translates Flint's format
  decisions into it; Go never parses d3 syntax.
- **Per-chart data shapes**: `Series.Values []DataPoint{X, Y}` stays for
  bar/line/scatter. Add `Series.OHLC []OHLCPoint{T, O, H, L, C}` and a
  top-level `Heat *HeatData` (cell list `{X, Y, Z}` or grid form). Scatter
  points gain optional `Size`/`Weight`.
- **Bar options**: `Orientation` (vertical/horizontal) and `Stacked bool`.
- **Theme**: add `Gradient []string` (ordered sequential stops) for heatmap
  colormaps, alongside categorical `Palette`.
- **`Build()` completion**: implement scaffolded types for tier 1 (line,
  scatter, heatmap), then tier 2 (OHLC, sparkline).

`ToECharts()` (web surface) is untouched — a bonus surface Flint does not
disturb.

### 2. TS Stage-3 backend (`@nimblemarkets/flint-ntcharts`)

Follows flint-chart backend conventions exactly: a `ChartTemplateDef`-per-chart
registry plus `assembleNtcharts(input: ChartAssemblyInput): NtchartsSpec`.
Consumes `ChannelSemantics` (Stage 1) and `LayoutResult` (Stage 2) — both
documented as backend-agnostic — and uses flint core's `pivot` machinery to
produce named series. Emits ntcharts-spec with everything pinned: sizes in
cells, Y domain, formats, colors.

- **px→cells**: `canvasSize`/`baseSize` are passed in cells and treated as
  Flint's abstract layout units directly, with terminal-tuned
  `AssembleOptions` (small `defaultBandSize`, tight `stepPadding`) and a ~2:1
  cell-aspect correction on height-derived decisions.
- **Chart types, tier 1**: Bar, Grouped Bar, Stacked Bar, Horizontal Bar,
  Line, Area (→ streamline fill), Scatter, Heatmap.
- **Tier 2**: Candlestick (OHLC), Sparkline.
- If any needed Stage-1/2 core API turns out not to be exported from
  `flint-chart`, submit a small export-only PR upstream.

### 3. Go compile package (`flint-ntcharts/compile`)

- `esbuild` bundles flint-chart + the backend into one JS file exporting
  `compile(inputJSON) → {spec, warnings, size}`.
- Javy compiles the bundle to `flint.wasm` (~1–2 MB), built in CI and
  embedded via `go:embed`.
- wazero executes it: pure Go, no cgo, no Node. Instances are pooled; each
  compile call is a pure function.

```go
func Compile(ctx context.Context, input []byte, opts ...Option) (spec.Spec, []Warning, error)
```

**Spike first** (phase 1): bundle today's flint-chart through Javy and compile
the gallery fixtures. Verify QuickJS feature coverage (watch: `Intl`, very new
ES features — esbuild targets ES2020) and that wasm output is byte-identical
to Node output.

### 4. `flint-tui` — live chart server (`flint-ntcharts/cmd/flint-tui`)

A bubbletea app acting as a persistent terminal chart window for agent
workflows:

- **Sources**: watch a file/glob, read NDJSON from stdin, or listen on a Unix
  socket. Agents (Claude Code hooks, MCP glue, scripts) push JSON; the pane
  live-updates to the newest chart.
- **Input sniffing**: flint `ChartAssemblyInput` (has `chart_spec`) → compile
  via embedded wasm; raw ntcharts-spec (has `type`) → render directly.
- **Resize-reactive**: on terminal resize, recompile with the new canvas size
  so Flint's layout engine redoes label thinning/sizing live.
- Compile errors/warnings render in a status line; they never kill the app.
- Multi-chart grid/tabs deferred to v2.

### 5. booba-shim `flintchart` (in `NimbleMarkets/booba-shim`)

Per the duckdb/pdfium pattern: a shim JS module (flint-chart browser ESM
bundle + a `compile` entry) installed by `booba-shim-assets`, and a Go package
exposing the **same signature as `flint-ntcharts/compile`** so app code swaps
implementations behind `//go:build js` tags. Browser Go-WASM TUIs then do:
compile via shim → `spec.Build()` → render in the go-booba terminal.

## Error handling

- Flint compile warnings (`_warnings`) travel in the compile envelope
  (`{spec, warnings, size}`) and surface in `flint-tui`'s status line.
- `spec.Validate()` remains the Go-side gate before `Build()`.
- Unknown/unsupported chart types fail with a clear error listing supported
  types (mirroring the existing scaffold behavior).

## Testing

- **TS**: run flint-chart's gallery fixtures (`test-data` generators) through
  `assembleNtcharts` → golden ntcharts-spec JSONs.
- **Go (ntcharts)**: golden `View()` string renders per spec fixture,
  extending the existing `// Output:` example-test style.
- **Go (flint-ntcharts)**: CI check that wasm compile output is
  byte-identical to Node compile output across the fixture set.
- **booba-shim**: browser smoke test per that repo's existing pattern.

## Phasing

1. **Javy spike** — bundle flint-chart, compile fixtures under wazero, verify
   QuickJS coverage and output parity. Go/no-go gate for the wasm approach
   (fallback: shell out to Node, same JSON contract).
2. **ntcharts-spec v1** — axis/format/data-shape evolution + `Build()` tier 1.
3. **TS backend tier 1** — eight chart types, golden-spec tests.
4. **Go compile package** — wasm embed, wazero runner, parity CI.
5. **flint-tui** — live chart server.
6. **booba-shim `flintchart`**.
7. **Tier 2 charts** — OHLC, Sparkline.
