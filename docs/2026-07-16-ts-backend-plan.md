# TypeScript Stage-3 Backend Implementation Plan (flint → ntcharts, Phase 3)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** An out-of-tree flint backend — `assembleNtcharts(input: ChartAssemblyInput)` in TypeScript — that consumes flint-chart's exported Stage-1/2 compiler and emits ntcharts-spec v1 JSON, for five tier-1 chart types, with golden tests and Go cross-validation (emitted JSON → `spec.Validate()` + `spec.Build()`).

**Architecture:** Follows `docs/adding-a-backend.md`'s assembly contract (pre-phase → semantics → zero → temporal → budgets/overflow → layout → instantiate), mirroring `assembleVegaLite`'s pipeline but consuming only the *exported* flint-chart API. The six unexported orchestration utilities get small local shims (`shims.ts`); color scheme names resolve through our own `colormap.ts` (needed regardless — flint carries scheme *names*, not hex). Templates split pivoted long-form rows into ntcharts-spec `Series` and emit fully-pinned specs (sizes in cells, Y domains, Go-layout time formats, hex colors).

**Tech Stack:** TypeScript, `flint-chart` (npm, ^0.2), esbuild (existing), vitest, Go 1.25 for the cross-validation harness.

**Design refs:** `docs/2026-07-15-flint-ntcharts-backend-design.md` §3; ntcharts-spec v1 = the `spec` package on the ntcharts repo's `spec` branch (see its README fidelity matrix and `spec.go` types).

## Global Constraints

- Work in `/Users/evan/projects/flint-ntcharts` (this repo, branch `main`). One task (Task 5 step for YAxis.Labels) also touches `/Users/evan/projects/ntcharts` branch `spec` — there, **NEVER `git add -A`/`git add .`/`git add -u`** (four go.mod/go.sum files carry uncommitted user changes); stage by explicit path only. In flint-ntcharts, explicit-path staging is also the rule (its go.mod WILL be edited by Task 6 — that's intended, stage it explicitly).
- The Phase-1 wasm spike must keep working: `js/src/compile.js`, `js/src/entry-javy.js`, `js/src/polyfills.js`, `js/scripts/node-runner.mjs`, and the `npm run build` script are **untouched** by this phase. `cd js && npm run build` must still succeed after every task.
- flint chart-type display names are exact strings: `"Bar Chart"`, `"Stacked Bar Chart"`, `"Line Chart"`, `"Scatter Plot"`, `"Heatmap"`. `"Grouped Bar Chart"` is intentionally NOT registered (ntcharts terminal cannot render grouped bars) — requesting it must produce the standard unknown-chart-type error listing supported types.
- ntcharts-spec invariants the emitter must honor (from Phase 2): `Format.Layout` is a **Go time layout**; `Format.Kind: "time"` values are **ms-since-epoch**; one-sided YAxis.Min/Max pins are legal (missing bound data-derived); multi-series bar requires `options.stacked: true`; heatmap data goes in `heat.cells` (numeric X/Y indices); gradients are ordered `#rrggbb` stops in `theme.gradient`; `omitzero` semantics — do not emit empty structs.
- Terminal sizing: `baseSize`/`canvasSize` are passed to flint in **cells**; `DEFAULT_TERMINAL_BASE = {width: 64, height: 20}`; terminal-tuned `AssembleOptions`: `{ minStep: 1, defaultBandSize: 3, stepPadding: 0.2 }`. Emitted `Spec.Width/Height` = `Math.max(8, Math.round(layout.subplotWidth))` / `Math.max(4, Math.round(layout.subplotHeight))`.
- Output convention mirrors flint backends: `assembleNtcharts` returns the ntcharts-spec object with private metadata keys `_warnings` (ChartWarning[], only when non-empty), `_width`, `_height` (cells). Go's `encoding/json` ignores unknown keys, so goldens keep them; the Go cross-validation strips nothing.
- Exported flint-chart API verified available (do not re-verify): `resolveChannelSemantics`, `computeZeroDecision`, `convertTemporalData`, `computeChannelBudgets`, `filterOverflow`, `computeLayout`, `applyPivot`, `makeCartesianPivot`, `applyEncodingOverrides`, `normalizeStaticSeries`, `detectBandedAxisFromSemantics`, `makeSortAction`, `computePaddedDomain`, `getRecommendedColorScheme`, and all types (`ChartAssemblyInput`, `ChartTemplateDef`, `InstantiateContext`, `ChannelSemantics`, `LayoutResult`, `AssembleOptions`, `ChartWarning`, `FormatSpec`, `LayoutDeclaration`). Confirmed UNexported (shim locally, do not import): `applyAggregation`, `normalizeChartProperties`, `resolveBaseSize`, `deriveStretchCaps`, `computeMinSubplotDimensions`, `decideColorMaps`.
- **Bounded-adaptation clause:** flint call signatures cited in this plan come from source at `/Users/evan/projects/flint-chart/packages/flint-js/src/`. If the installed package's `node_modules/flint-chart/dist/index.d.ts` differs in arg order/optionality, adapt to the `.d.ts` (it is the contract) and record the delta in your report.
- Facets, aggregation beyond sum/average/count/min/max/median, and the `"Grouped Bar Chart"`/`"Area Chart"`/`"Sparkline"`/`"Candlestick Chart"` types are OUT of scope (tier 2+).
- Test commands: `cd js && npx vitest run` (TS) and `go test ./speccheck/` (Go, Task 6). Goldens regenerate via `UPDATE_GOLDEN=1 npx vitest run`.

---

### Task 1: TypeScript toolchain + NtSpec types + skeleton

**Files:**
- Modify: `js/package.json` (name → `@nimblemarkets/flint-ntcharts`, add devDeps typescript/vitest/@types/node, add scripts)
- Create: `js/tsconfig.json`
- Create: `js/src/ntcharts/types.ts`
- Create: `js/src/ntcharts/index.ts`
- Create: `js/src/ntcharts/assemble.ts` (skeleton)
- Create: `js/test/skeleton.test.ts`

**Interfaces:**
- Produces: the `NtSpec`/`NtSeries`/`NtFormat`/… TS types every later task uses (exact mirror of ntcharts-spec v1 JSON), and `assembleNtcharts(input): NtSpecOut` skeleton that throws `Unknown chart type "<t>". Supported: <list>` for everything (registry empty).

- [ ] **Step 1: Write the failing test**

`js/test/skeleton.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { assembleNtcharts } from "../src/ntcharts/index.js";

describe("assembleNtcharts skeleton", () => {
  it("rejects unknown chart types with the supported list", () => {
    expect(() =>
      assembleNtcharts({
        data: { values: [{ a: 1, b: 2 }] },
        chart_spec: { chartType: "Rose Chart", encodings: { x: { field: "a" }, y: { field: "b" } } },
      } as any),
    ).toThrow(/Unknown chart type "Rose Chart"/);
  });
});
```

- [ ] **Step 2: Set up the toolchain and run the test to verify it fails**

`js/package.json` changes: `"name": "@nimblemarkets/flint-ntcharts"`, keep `"private": true` and `"type": "module"`; add scripts `"test": "vitest run"`, `"typecheck": "tsc --noEmit"`; `npm install -D typescript vitest @types/node`.

`js/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "strict": true,
    "noEmit": true,
    "allowJs": true,
    "skipLibCheck": true
  },
  "include": ["src/ntcharts/**/*.ts", "test/**/*.ts"]
}
```

Run: `cd /Users/evan/projects/flint-ntcharts/js && npx vitest run`
Expected: FAIL — cannot resolve `../src/ntcharts/index.js`.

- [ ] **Step 3: Write the NtSpec types**

`js/src/ntcharts/types.ts` — exact mirror of ntcharts-spec v1 JSON tags (see the ntcharts repo `spec/spec.go` on branch `spec`; json tags are authoritative):

```ts
// ntcharts-spec v1 wire types. Mirrors github.com/NimbleMarkets/ntcharts/v2/spec
// (branch `spec`). Field names are the Go structs' json tags. Optional fields
// use undefined (never null) so JSON.stringify omits them, matching omitzero.

export interface NtFormat {
  kind?: "number" | "percent" | "currency" | "si" | "time";
  precision?: number;
  currency?: string;
  layout?: string; // Go time layout
}

export interface NtXAxis {
  title?: string;
  type?: "category" | "time" | "value";
  labels?: string[];
  format?: NtFormat;
}

export interface NtYAxis {
  title?: string;
  min?: number;
  max?: number;
  labels?: string[]; // added for heatmap row labels (Task 5)
  format?: NtFormat;
}

export interface NtDataPoint {
  x?: unknown; // string | number | RFC3339 string | ms-since-epoch
  y: number;
  size?: number;
}

export interface NtOHLCPoint { t: unknown; o: number; h: number; l: number; c: number; }

export interface NtSeries {
  name: string;
  type?: string;
  values?: NtDataPoint[];
  ohlc?: NtOHLCPoint[];
  color?: string;
}

export interface NtHeatCell { x: number; y: number; z: number; }
export interface NtHeatData {
  cells?: NtHeatCell[];
  matrix?: number[][];
  min_value?: number;
  max_value?: number;
}

export interface NtOptions {
  show_legend?: boolean;
  show_grid?: boolean;
  orientation?: "vertical" | "horizontal";
  stacked?: boolean;
}

export interface NtTheme {
  background?: string;
  foreground?: string;
  palette?: string[];
  gradient?: string[];
}

export interface NtData { series: NtSeries[]; x_axis_data?: unknown[]; }

export interface NtSpec {
  type: "bar" | "line" | "timeseries" | "streamline" | "sparkline" | "heatmap" | "ohlc" | "scatter" | "canvas";
  title?: string;
  subtitle?: string;
  width: number;
  height: number;
  x_axis?: NtXAxis;
  y_axis?: NtYAxis;
  data: NtData;
  heat?: NtHeatData;
  options?: NtOptions;
  theme?: NtTheme;
}

import type { ChartWarning } from "flint-chart";
export type NtSpecOut = NtSpec & { _warnings?: ChartWarning[]; _width?: number; _height?: number };
```

- [ ] **Step 4: Skeleton assemble + barrel; test passes; spike still builds**

`js/src/ntcharts/assemble.ts` (skeleton for now):

```ts
import type { ChartAssemblyInput } from "flint-chart";
import type { NtSpecOut } from "./types.js";
import { ntGetTemplateDef, ntSupportedChartTypes } from "./templates/index.js";

export function assembleNtcharts(input: ChartAssemblyInput): NtSpecOut {
  const chartType = input.chart_spec.chartType;
  const def = ntGetTemplateDef(chartType);
  if (!def) {
    throw new Error(
      `Unknown chart type "${chartType}". Supported: ${ntSupportedChartTypes().join(", ")}`,
    );
  }
  throw new Error("assembleNtcharts: pipeline not implemented yet (Task 3)");
}
```

Create `js/src/ntcharts/templates/index.ts` with an empty registry:

```ts
import type { ChartTemplateDef } from "flint-chart";

const defs: ChartTemplateDef[] = [];

export function ntAllTemplateDefs(): ChartTemplateDef[] { return defs; }
export function ntGetTemplateDef(chart: string): ChartTemplateDef | undefined {
  return defs.find((d) => d.chart === chart);
}
export function ntSupportedChartTypes(): string[] { return defs.map((d) => d.chart); }
export function ntRegister(def: ChartTemplateDef): void { defs.push(def); }
```

`js/src/ntcharts/index.ts`: `export { assembleNtcharts } from "./assemble.js"; export * from "./types.js"; export { ntAllTemplateDefs, ntGetTemplateDef, ntSupportedChartTypes } from "./templates/index.js";`

Run: `npx vitest run` → PASS. `npx tsc --noEmit` → clean. `npm run build` (spike bundle) → still succeeds.

- [ ] **Step 5: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add js/package.json js/package-lock.json js/tsconfig.json js/src/ntcharts/ js/test/
git commit -m "feat(ts): package rename, TS toolchain, NtSpec types, assemble skeleton"
```

---

### Task 2: shims, format translation, colormap (pure functions + unit tests)

**Files:**
- Create: `js/src/ntcharts/shims.ts`, `js/src/ntcharts/format.ts`, `js/src/ntcharts/colormap.ts`
- Create: `js/test/format.test.ts`, `js/test/colormap.test.ts`, `js/test/shims.test.ts`

**Interfaces:**
- Produces (consumed by Tasks 3–5):
  - `resolveBaseSizeShim(base, ceiling): {width,height}` — default `{width:64,height:20}` (cells), clamped to ceiling per-dimension.
  - `deriveStretchCapsShim(base, ceiling, maxStretch=1.5): {maxStretchX,maxStretchY}` — `max(1, ceiling/base)` per dimension, `maxStretch` fallback when no ceiling.
  - `applyAggregationShim(encodings, rows): rows` — group rows by all non-aggregate encoded fields; apply `sum|average|count|min|max|median` to aggregate-encoded fields; rows pass through when no encoding declares `aggregate`.
  - `formatSpecToNt(fmt: FormatSpec|undefined, warn: (w)=>void): NtFormat|undefined` and `d3TimeToGoLayout(d3fmt: string, warn): string`.
  - `paletteForScheme(scheme: string|undefined): string[]` (categorical hex), `gradientForScheme(scheme: string|undefined): string[]` (ordered stops).

- [ ] **Step 1: Write the failing tests**

`js/test/format.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { formatSpecToNt, d3TimeToGoLayout } from "../src/ntcharts/format.js";

const noWarn = () => {};

describe("formatSpecToNt", () => {
  it("maps currency prefix", () => {
    expect(formatSpecToNt({ pattern: ",.2f", prefix: "$" }, noWarn)).toEqual({
      kind: "currency", currency: "$", precision: 2,
    });
  });
  it("maps d3 percent patterns", () => {
    expect(formatSpecToNt({ pattern: ".0~%" }, noWarn)).toEqual({ kind: "percent", precision: 0 });
    expect(formatSpecToNt({ pattern: ".1%" }, noWarn)).toEqual({ kind: "percent", precision: 1 });
  });
  it("maps abbreviate to si", () => {
    expect(formatSpecToNt({ abbreviate: true }, noWarn)).toEqual({ kind: "si", precision: 1 });
  });
  it("maps plain decimal patterns to number+precision", () => {
    expect(formatSpecToNt({ pattern: ",.3f" }, noWarn)).toEqual({ kind: "number", precision: 3 });
    expect(formatSpecToNt({ pattern: ",d" }, noWarn)).toEqual({ kind: "number", precision: 0 });
  });
  it("drops suffix with a warning and returns undefined for empty", () => {
    const warns: unknown[] = [];
    expect(formatSpecToNt({ suffix: "°C" }, (w) => warns.push(w))).toBeUndefined();
    expect(warns).toHaveLength(1);
    expect(formatSpecToNt({}, noWarn)).toBeUndefined();
    expect(formatSpecToNt(undefined, noWarn)).toBeUndefined();
  });
});

describe("d3TimeToGoLayout", () => {
  it("translates common tokens", () => {
    expect(d3TimeToGoLayout("%Y-%m-%d", noWarn)).toBe("2006-01-02");
    expect(d3TimeToGoLayout("%b %d", noWarn)).toBe("Jan 02");
    expect(d3TimeToGoLayout("%H:%M", noWarn)).toBe("15:04");
    expect(d3TimeToGoLayout("%Y", noWarn)).toBe("2006");
  });
  it("warns on untranslatable tokens and passes them through", () => {
    const warns: unknown[] = [];
    expect(d3TimeToGoLayout("%Y w%U", (w) => warns.push(w))).toBe("2006 w%U");
    expect(warns).toHaveLength(1);
  });
});
```

`js/test/colormap.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { paletteForScheme, gradientForScheme } from "../src/ntcharts/colormap.js";

describe("colormap", () => {
  it("returns tableau10 for the default categorical scheme", () => {
    const p = paletteForScheme("tableau10");
    expect(p[0]).toBe("#4e79a7");
    expect(p).toHaveLength(10);
  });
  it("falls back to tableau10 for unknown categorical schemes", () => {
    expect(paletteForScheme("no-such-scheme")).toEqual(paletteForScheme("tableau10"));
    expect(paletteForScheme(undefined)).toEqual(paletteForScheme("tableau10"));
  });
  it("returns ordered hex stops for sequential schemes", () => {
    const g = gradientForScheme("viridis");
    expect(g.length).toBeGreaterThanOrEqual(5);
    for (const stop of g) expect(stop).toMatch(/^#[0-9a-f]{6}$/);
  });
  it("falls back to viridis for unknown sequential schemes", () => {
    expect(gradientForScheme("mystery")).toEqual(gradientForScheme("viridis"));
  });
});
```

`js/test/shims.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { resolveBaseSizeShim, deriveStretchCapsShim, applyAggregationShim } from "../src/ntcharts/shims.js";

describe("shims", () => {
  it("resolveBaseSizeShim defaults and clamps to ceiling", () => {
    expect(resolveBaseSizeShim(undefined, undefined)).toEqual({ width: 64, height: 20 });
    expect(resolveBaseSizeShim({ width: 100, height: 30 }, { width: 80, height: 40 }))
      .toEqual({ width: 80, height: 30 });
  });
  it("deriveStretchCapsShim ratios ≥ 1", () => {
    expect(deriveStretchCapsShim({ width: 64, height: 20 }, { width: 128, height: 20 }))
      .toEqual({ maxStretchX: 2, maxStretchY: 1 });
    expect(deriveStretchCapsShim({ width: 64, height: 20 }, undefined))
      .toEqual({ maxStretchX: 1.5, maxStretchY: 1.5 });
  });
  it("applyAggregationShim sums grouped rows and passes through otherwise", () => {
    const rows = [
      { cat: "a", v: 1 }, { cat: "a", v: 2 }, { cat: "b", v: 5 },
    ];
    const encodings = { x: { field: "cat" }, y: { field: "v", aggregate: "sum" } };
    expect(applyAggregationShim(encodings as any, rows)).toEqual([
      { cat: "a", v: 3 }, { cat: "b", v: 5 },
    ]);
    const plain = { x: { field: "cat" }, y: { field: "v" } };
    expect(applyAggregationShim(plain as any, rows)).toEqual(rows);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail** (`npx vitest run` → module-not-found failures)

- [ ] **Step 3: Implement the three modules**

`js/src/ntcharts/format.ts`:

```ts
import type { FormatSpec, ChartWarning } from "flint-chart";
import type { NtFormat } from "./types.js";

type Warn = (w: ChartWarning) => void;

// d3 pattern precision: ".2f" / ",.2f" / ".0~%" → the digit run after the dot.
function patternPrecision(pattern: string, fallback: number): number {
  const m = /\.(\d+)/.exec(pattern);
  return m ? parseInt(m[1], 10) : fallback;
}

// formatSpecToNt translates flint's FormatSpec (d3-format struct) into the
// ntcharts-spec Format directive. Suffixes have no spec equivalent and are
// dropped with a warning. Returns undefined when nothing meaningful is set,
// so callers omit the field (omitzero parity).
export function formatSpecToNt(fmt: FormatSpec | undefined, warn: Warn): NtFormat | undefined {
  if (!fmt) return undefined;
  const { pattern, prefix, suffix, abbreviate } = fmt;
  if (suffix) {
    warn({ severity: "info", code: "format-suffix-dropped",
      message: `ntcharts-spec has no format suffix; dropping ${JSON.stringify(suffix)}` });
  }
  if (prefix) {
    return { kind: "currency", currency: prefix, precision: patternPrecision(pattern ?? "", 2) };
  }
  if (pattern && pattern.includes("%")) {
    return { kind: "percent", precision: patternPrecision(pattern, 0) };
  }
  if (abbreviate) return { kind: "si", precision: 1 };
  if (pattern) {
    if (/d$/.test(pattern)) return { kind: "number", precision: 0 };
    return { kind: "number", precision: patternPrecision(pattern, 2) };
  }
  return undefined;
}

const D3_TO_GO: Array<[string, string]> = [
  ["%Y", "2006"], ["%y", "06"], ["%B", "January"], ["%b", "Jan"],
  ["%m", "01"], ["%A", "Monday"], ["%a", "Mon"], ["%d", "02"], ["%e", "_2"],
  ["%H", "15"], ["%I", "03"], ["%M", "04"], ["%S", "05"], ["%p", "PM"], ["%Z", "MST"],
];

// d3TimeToGoLayout best-effort-translates a d3 time-format string into a Go
// time layout. Unknown %-tokens pass through verbatim with one warning.
export function d3TimeToGoLayout(d3fmt: string, warn: Warn): string {
  let out = d3fmt;
  for (const [from, to] of D3_TO_GO) out = out.split(from).join(to);
  if (/%[A-Za-z]/.test(out)) {
    warn({ severity: "info", code: "time-format-partial",
      message: `d3 time format ${JSON.stringify(d3fmt)} contains untranslatable tokens; passed through` });
  }
  return out;
}
```

`js/src/ntcharts/colormap.ts`:

```ts
// Scheme-name → concrete colors. flint's ChannelSemantics.colorScheme carries
// Vega scheme NAMES; concrete palette selection is each backend's job.
const CATEGORICAL: Record<string, string[]> = {
  tableau10: ["#4e79a7", "#f28e2c", "#e15759", "#76b7b2", "#59a14f",
              "#edc949", "#af7aa1", "#ff9da7", "#9c755f", "#bab0ab"],
  set1: ["#e41a1c", "#377eb8", "#4daf4a", "#984ea3", "#ff7f00",
         "#ffff33", "#a65628", "#f781bf", "#999999"],
  set2: ["#66c2a5", "#fc8d62", "#8da0cb", "#e78ac3", "#a6d854",
         "#ffd92f", "#e5c494", "#b3b3b3"],
  tableau20: ["#4e79a7", "#a0cbe8", "#f28e2c", "#ffbe7d", "#59a14f",
              "#8cd17d", "#b6992d", "#f1ce63", "#499894", "#86bcb6",
              "#e15759", "#ff9d9a", "#79706e", "#bab0ab", "#d37295",
              "#fabfd2", "#b07aa1", "#d4a6c8", "#9d7660", "#d7b5a6"],
};

const SEQUENTIAL: Record<string, string[]> = {
  viridis: ["#440154", "#414487", "#2a788e", "#22a884", "#7ad151", "#fde725"],
  blues:   ["#f7fbff", "#c6dbef", "#6baed6", "#2171b5", "#08306b"],
  reds:    ["#fff5f0", "#fcbba1", "#fb6a4a", "#cb181d", "#67000d"],
  oranges: ["#fff5eb", "#fdd0a2", "#fd8d3c", "#d94801", "#7f2704"],
  purples: ["#fcfbfd", "#dadaeb", "#9e9ac8", "#6a51a3", "#3f007d"],
  yelloworangebrown: ["#ffffe5", "#fee391", "#fe9929", "#cc4c02", "#662506"],
  redblue: ["#67001f", "#d6604d", "#f7f7f7", "#4393c3", "#053061"], // diverging
};

export function paletteForScheme(scheme: string | undefined): string[] {
  return CATEGORICAL[scheme ?? ""] ?? CATEGORICAL.tableau10;
}

export function gradientForScheme(scheme: string | undefined): string[] {
  return SEQUENTIAL[scheme ?? ""] ?? SEQUENTIAL.viridis;
}
```

`js/src/ntcharts/shims.ts`:

```ts
// Local stand-ins for flint-chart core utilities that exist upstream but are
// not exported from the npm package (applyAggregation, resolveBaseSize,
// deriveStretchCaps). Semantics documented from the upstream source; if an
// upstream release exports them, delete this module and import instead.
import type { ChartEncoding } from "flint-chart";

export interface Size { width: number; height: number; }

export const DEFAULT_TERMINAL_BASE: Size = { width: 64, height: 20 };

export function resolveBaseSizeShim(base: Size | undefined, ceiling: Size | undefined): Size {
  const b = { ...(base ?? DEFAULT_TERMINAL_BASE) };
  if (ceiling) {
    b.width = Math.min(b.width, ceiling.width);
    b.height = Math.min(b.height, ceiling.height);
  }
  return b;
}

export function deriveStretchCapsShim(
  base: Size, ceiling: Size | undefined, maxStretch = 1.5,
): { maxStretchX: number; maxStretchY: number } {
  if (!ceiling) return { maxStretchX: maxStretch, maxStretchY: maxStretch };
  return {
    maxStretchX: Math.max(1, ceiling.width / base.width),
    maxStretchY: Math.max(1, ceiling.height / base.height),
  };
}

type Row = Record<string, unknown>;
const AGGS: Record<string, (vals: number[]) => number> = {
  sum: (v) => v.reduce((a, b) => a + b, 0),
  average: (v) => v.reduce((a, b) => a + b, 0) / v.length,
  count: (v) => v.length,
  min: (v) => Math.min(...v),
  max: (v) => Math.max(...v),
  median: (v) => { const s = [...v].sort((a, b) => a - b); const m = s.length >> 1;
    return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2; },
};

// applyAggregationShim mirrors upstream core/aggregate.ts behavior: when any
// encoding declares `aggregate`, group rows by every OTHER encoded field and
// collapse each aggregate-encoded field with its function.
export function applyAggregationShim(
  encodings: Record<string, ChartEncoding | undefined>, rows: Row[],
): Row[] {
  const entries = Object.values(encodings).filter((e): e is ChartEncoding => !!e && !!(e as any).field);
  const aggFields = entries.filter((e) => (e as any).aggregate);
  if (aggFields.length === 0) return rows;
  const groupFields = entries.filter((e) => !(e as any).aggregate).map((e) => (e as any).field as string);
  const groups = new Map<string, Row[]>();
  for (const row of rows) {
    const key = JSON.stringify(groupFields.map((f) => row[f]));
    (groups.get(key) ?? groups.set(key, []).get(key)!).push(row);
  }
  const out: Row[] = [];
  for (const bucket of groups.values()) {
    const merged: Row = { ...bucket[0] };
    for (const enc of aggFields) {
      const field = (enc as any).field as string;
      const fn = AGGS[(enc as any).aggregate as string] ?? AGGS.sum;
      merged[field] = fn(bucket.map((r) => Number(r[field])).filter((n) => !Number.isNaN(n)));
    }
    out.push(merged);
  }
  return out;
}
```

- [ ] **Step 4: Run tests to verify they pass** (`npx vitest run` + `npx tsc --noEmit`)

- [ ] **Step 5: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add js/src/ntcharts/shims.ts js/src/ntcharts/format.ts js/src/ntcharts/colormap.ts js/test/format.test.ts js/test/colormap.test.ts js/test/shims.test.ts
git commit -m "feat(ts): shims, format translation (d3→Go), scheme colormaps"
```

---

### Task 3: assemble pipeline + series splitting + Bar Chart template + first golden

**Files:**
- Modify: `js/src/ntcharts/assemble.ts` (full pipeline)
- Create: `js/src/ntcharts/series.ts`
- Create: `js/src/ntcharts/templates/bar.ts`
- Modify: `js/src/ntcharts/templates/index.ts` (register bar)
- Create: `js/test/golden.test.ts`, `js/test/helpers.ts`
- Create (generated): `testdata/ntspec-golden/bar-currency.json`

**Interfaces:**
- Consumes: Task 2 modules; exported flint-chart pipeline functions.
- Produces: the full `assembleNtcharts` orchestrator; `NtInstantiateContext` (extends flint's `InstantiateContext` with `emit: NtSpec`, `warn(w)`, `ntFormatX/ntFormatY`); `splitSeries(ctx): NtSeries[]`; the golden-test harness (`assembleFixture(name)`, `expectGolden(name, out)` with `UPDATE_GOLDEN=1` regeneration).

- [ ] **Step 1: Write the failing golden test + harness**

`js/test/helpers.ts`:

```ts
import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { assembleNtcharts } from "../src/ntcharts/index.js";

const fixtureDir = fileURLToPath(new URL("../../testdata/fixtures/", import.meta.url));
const goldenDir = fileURLToPath(new URL("../../testdata/ntspec-golden/", import.meta.url));

export function assembleFixture(name: string) {
  const input = JSON.parse(readFileSync(`${fixtureDir}${name}.json`, "utf8"));
  return assembleNtcharts(input);
}

export function expectGolden(name: string, out: unknown): void {
  mkdirSync(goldenDir, { recursive: true });
  const path = `${goldenDir}${name}.json`;
  const rendered = JSON.stringify(out, null, 2) + "\n";
  if (process.env.UPDATE_GOLDEN === "1" || !existsSync(path)) {
    writeFileSync(path, rendered);
    return;
  }
  const expected = readFileSync(path, "utf8");
  if (rendered !== expected) {
    throw new Error(`golden mismatch for ${name}; run UPDATE_GOLDEN=1 to regenerate\n--- got ---\n${rendered.slice(0, 2000)}`);
  }
}
```

`js/test/golden.test.ts` (bar only for this task):

```ts
import { describe, it, expect } from "vitest";
import { assembleFixture, expectGolden } from "./helpers.js";

describe("golden: bar-currency", () => {
  it("emits a stacked=false single-series bar spec with currency Y format", () => {
    const out = assembleFixture("bar-currency");
    expect(out.type).toBe("bar");
    expect(out.data.series).toHaveLength(1);
    expect(out.x_axis?.labels).toContain("Widgets");
    expect(out.y_axis?.format?.kind).toBe("currency");
    expect(out.y_axis?.min).toBe(0); // bar zero decision
    expect(out.width).toBeGreaterThanOrEqual(8);
    expect(out.height).toBeGreaterThanOrEqual(4);
    expectGolden("bar-currency", out);
  });
});
```

Run `npx vitest run` → FAIL (pipeline not implemented).

- [ ] **Step 2: Implement the orchestrator**

`js/src/ntcharts/assemble.ts` — mirror `assembleVegaLite`'s call sequence using ONLY exported functions + shims. Full implementation:

```ts
import {
  resolveChannelSemantics, computeZeroDecision, convertTemporalData,
  computeChannelBudgets, filterOverflow, computeLayout,
  applyPivot, applyEncodingOverrides, normalizeStaticSeries,
} from "flint-chart";
import type {
  ChartAssemblyInput, ChartTemplateDef, ChannelSemantics, ChartWarning,
  LayoutResult, AssembleOptions, LayoutDeclaration,
} from "flint-chart";
import { ntGetTemplateDef, ntSupportedChartTypes } from "./templates/index.js";
import { resolveBaseSizeShim, deriveStretchCapsShim, applyAggregationShim } from "./shims.js";
import { formatSpecToNt, d3TimeToGoLayout } from "./format.js";
import type { NtSpec, NtSpecOut, NtFormat } from "./types.js";

const TERMINAL_OPTIONS: AssembleOptions = { minStep: 1, defaultBandSize: 3, stepPadding: 0.2 };

export interface NtInstantiateContext {
  channelSemantics: Record<string, ChannelSemantics>;
  layout: LayoutResult;
  table: Record<string, unknown>[];
  encodings: Record<string, any>;
  chartProperties?: Record<string, unknown>;
  canvasSize: { width: number; height: number };
  chartType: string;
  emit: NtSpec;               // template fills this in
  warn: (w: ChartWarning) => void;
  ntFormatX?: NtFormat;       // pre-translated axis formats
  ntFormatY?: NtFormat;
}

export function assembleNtcharts(input: ChartAssemblyInput): NtSpecOut {
  const warnings: ChartWarning[] = [];
  const warn = (w: ChartWarning) => warnings.push(w);
  const chartType = input.chart_spec.chartType;
  let template: ChartTemplateDef | undefined = ntGetTemplateDef(chartType);
  if (!template) {
    throw new Error(`Unknown chart type "${chartType}". Supported: ${ntSupportedChartTypes().join(", ")}`);
  }
  const semanticTypes = input.semantic_types ?? {};
  const rawData: Record<string, unknown>[] = (input.data as any).values ?? [];

  // Sizes: interpreted in CELLS throughout.
  const ceiling = input.chart_spec.canvasSize;
  const baseSize = resolveBaseSizeShim(input.chart_spec.baseSize, ceiling);
  const caps = deriveStretchCapsShim(baseSize, ceiling);

  // PRE-PHASE
  const normalized = normalizeStaticSeries(input.chart_spec.encodings ?? {}, rawData, semanticTypes);
  let data = normalized.data;

  // Preliminary typing so pivot/overrides see resolved types.
  const prelimConverted = convertTemporalData(data, semanticTypes);
  const prelim = resolveChannelSemantics(normalized.encodings, data, semanticTypes, prelimConverted);
  const typedEncodings: Record<string, any> = {};
  for (const [ch, enc] of Object.entries(normalized.encodings)) {
    if (!enc) continue;
    typedEncodings[ch] = { ...(enc as object), type: (enc as any).type ?? prelim[ch]?.type };
  }

  // Pivot (may re-dispatch chart type within our registry).
  const pivoted = applyPivot(template, typedEncodings, data, input.chart_spec.chartProperties, ntGetTemplateDef);
  if (pivoted.chartType && pivoted.chartType !== template.chart) {
    const next = ntGetTemplateDef(pivoted.chartType);
    if (next) template = next;
  }
  const composed = applyEncodingOverrides(template, pivoted.encodings, input.chart_spec.chartProperties);
  const encodings = template.normalizeEncodings?.(composed, data) ?? composed;
  data = applyAggregationShim(encodings, data);

  // PHASE 0: semantics + zero.
  const convertedData = convertTemporalData(data, semanticTypes);
  const channelSemantics = resolveChannelSemantics(encodings, data, semanticTypes, convertedData);
  const markType = (template.template as any)?.mark ?? "point";
  for (const axis of ["x", "y"] as const) {
    const cs = channelSemantics[axis];
    if (cs?.type === "quantitative") {
      const nums = convertedData.map((r: any) => Number(r[cs.field])).filter((n: number) => !Number.isNaN(n));
      cs.zero = computeZeroDecision(cs.semanticAnnotation.semanticType, axis, markType, nums);
    }
  }

  // STEP 0a/0c + PHASE 1: layout.
  const declaration: LayoutDeclaration =
    template.declareLayoutMode?.(channelSemantics, data, input.chart_spec.chartProperties) ?? {};
  const options: AssembleOptions = { ...TERMINAL_OPTIONS, ...declaration.paramOverrides, ...caps };
  const budgets = computeChannelBudgets(channelSemantics, declaration, convertedData, baseSize, options);
  const overflow = filterOverflow(channelSemantics, declaration, encodings, convertedData, budgets, [markType]);
  warnings.push(...((overflow as any).warnings ?? []), ...((overflow as any).truncations ?? []));
  const layout = computeLayout(channelSemantics, declaration, overflow.values, baseSize, options, budgets.facetGrid);

  // Axis formats (translated once; templates place them).
  const ntFormatX = axisFormat(channelSemantics.x, warn);
  const ntFormatY = axisFormat(channelSemantics.y, warn);

  // PHASE 2: instantiate into an NtSpec.
  const emit: NtSpec = {
    type: "bar", // template overwrites
    width: Math.max(8, Math.round(layout.subplotWidth)),
    height: Math.max(4, Math.round(layout.subplotHeight)),
    data: { series: [] },
  };
  if (input.chart_spec.title) emit.title = String(input.chart_spec.title);
  const ctx: NtInstantiateContext = {
    channelSemantics, layout, table: overflow.values, encodings,
    chartProperties: input.chart_spec.chartProperties, canvasSize: baseSize,
    chartType: template.chart, emit, warn, ntFormatX, ntFormatY,
  };
  template.instantiate(emit as any, ctx as any);

  const result: NtSpecOut = { ...emit };
  if (warnings.length) result._warnings = warnings;
  result._width = emit.width;
  result._height = emit.height;
  return result;
}

function axisFormat(cs: ChannelSemantics | undefined, warn: (w: ChartWarning) => void): NtFormat | undefined {
  if (!cs) return undefined;
  if (cs.type === "temporal" && cs.temporalFormat) {
    return { kind: "time", layout: d3TimeToGoLayout(cs.temporalFormat, warn) };
  }
  return formatSpecToNt(cs.format, warn);
}
```

Note on `chart_spec.title`: if `ChartAssemblyInput` has no title field in the `.d.ts`, drop that line (bounded adaptation) — do not invent one.

- [ ] **Step 3: Implement series splitting**

`js/src/ntcharts/series.ts`:

```ts
import type { ChannelSemantics } from "flint-chart";
import { paletteForScheme } from "./colormap.js";
import type { NtSeries, NtDataPoint } from "./types.js";

// splitSeries groups long-form rows into ntcharts-spec Series by the color
// (or group) channel's field value. Without a grouping channel, one series
// named after the y field is produced. Colors come from the recommended
// categorical scheme, in first-appearance order (respecting ordinalSortOrder
// when flint resolved one).
export function splitSeries(
  table: Record<string, unknown>[],
  channelSemantics: Record<string, ChannelSemantics>,
  pointOf: (row: Record<string, unknown>) => NtDataPoint,
): NtSeries[] {
  const groupCS = channelSemantics.color ?? channelSemantics.group;
  const yField = channelSemantics.y?.field ?? "value";
  if (!groupCS || groupCS.type === "quantitative") {
    return [{ name: String(yField), values: table.map(pointOf) }];
  }
  const order: string[] = [];
  const buckets = new Map<string, NtDataPoint[]>();
  const preferred = groupCS.ordinalSortOrder;
  if (preferred) for (const v of preferred) { order.push(String(v)); buckets.set(String(v), []); }
  for (const row of table) {
    const key = String(row[groupCS.field]);
    if (!buckets.has(key)) { order.push(key); buckets.set(key, []); }
    buckets.get(key)!.push(pointOf(row));
  }
  const palette = paletteForScheme(groupCS.colorScheme?.scheme);
  return order
    .filter((name) => (buckets.get(name) ?? []).length > 0)
    .map((name, i) => ({ name, values: buckets.get(name)!, color: palette[i % palette.length] }));
}
```

- [ ] **Step 4: Implement the Bar Chart template**

`js/src/ntcharts/templates/bar.ts`:

```ts
import type { ChartTemplateDef } from "flint-chart";
import { detectBandedAxisFromSemantics, makeCartesianPivot, makeSortAction } from "flint-chart";
import { splitSeries } from "../series.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntBarChartDef: ChartTemplateDef = {
  chart: "Bar Chart",
  template: { mark: "bar" },
  channels: ["x", "y", "color", "group"],
  markCognitiveChannel: "length",
  declareLayoutMode: (cs, table) => {
    const result = detectBandedAxisFromSemantics(cs, table, { preferAxis: "x" });
    return {
      axisFlags: result ? { [result.axis]: { banded: true } } : { x: { banded: true } },
      resolvedTypes: result?.resolvedTypes,
    };
  },
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    // Orientation: banded/nominal axis on y ⇒ horizontal bars.
    const horizontal = cs.y != null && cs.y.type !== "quantitative" && cs.x?.type === "quantitative";
    const catCS = horizontal ? cs.y! : cs.x!;
    const valCS = horizontal ? cs.x! : cs.y!;

    emit.type = "bar";
    if (horizontal) emit.options = { ...emit.options, orientation: "horizontal" };

    // Category labels in table order (post-overflow), deduped.
    const labels: string[] = [];
    const seen = new Set<string>();
    for (const row of table) {
      const v = String(row[catCS.field]);
      if (!seen.has(v)) { seen.add(v); labels.push(v); }
    }
    emit.x_axis = { ...emit.x_axis, type: "category", labels, title: catCS.field };
    emit.y_axis = { ...emit.y_axis, title: valCS.field };
    const valueFormat = horizontal ? ctx.ntFormatX : ctx.ntFormatY;
    if (valueFormat) emit.y_axis.format = valueFormat;
    if (valCS.zero?.zero) emit.y_axis.min = 0;

    // Values aligned to the label order; series split by color.
    const index = new Map(labels.map((l, i) => [l, i]));
    const series = splitSeries(table, cs, (row) => ({ y: Number(row[valCS.field]) }));
    for (const s of series) {
      const slots = labels.map(() => ({ y: 0 }));
      for (const row of table) {
        const cat = String(row[catCS.field]);
        const belongs = series.length === 1 || !cs.color || String(row[cs.color.field]) === s.name;
        if (belongs) slots[index.get(cat)!] = { y: Number(row[valCS.field]) };
      }
      s.values = slots;
    }
    emit.data.series = series;
    if (series.length > 1) emit.options = { ...emit.options, stacked: true, show_legend: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({
    transpose: [["x", "y"]],
    permute: [["x", "y", "color"]],
    shift: ["color", "group"],
  }),
};
```

Register in `templates/index.ts`: `import { ntBarChartDef } from "./bar.js"; ntRegister(ntBarChartDef);` (convert registry to static list rather than mutate-on-import if cleaner — keep `ntGetTemplateDef` behavior identical).

- [ ] **Step 5: Run, capture golden, verify assertions, commit**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npx vitest run && npx tsc --noEmit && npm run build
git -C .. add js/src/ntcharts/ js/test/golden.test.ts js/test/helpers.ts testdata/ntspec-golden/bar-currency.json
git -C .. commit -m "feat(ts): assemble pipeline, series splitting, Bar Chart template + golden"
```

Read the generated `testdata/ntspec-golden/bar-currency.json` before committing: labels must be the four product names, one series, y_axis currency format, min 0, plausible cell dimensions. If flint call signatures forced adaptations, record them.

---

### Task 4: Stacked Bar + Line templates + goldens

**Files:**
- Create: `js/src/ntcharts/templates/stacked-bar.ts`, `js/src/ntcharts/templates/line.ts`
- Modify: `js/src/ntcharts/templates/index.ts`
- Modify: `js/test/golden.test.ts` (add cases)
- Create (generated): `testdata/ntspec-golden/stacked-bar.json`, `testdata/ntspec-golden/line-temporal.json`

**Interfaces:**
- Consumes: Task 3 orchestrator/ctx/splitSeries.
- Produces: `"Stacked Bar Chart"` and `"Line Chart"` registered. Line dispatch rule: x temporal ⇒ `type: "timeseries"` with X = **ms-since-epoch numbers**; x quantitative ⇒ `type: "line"` with numeric X.

- [ ] **Step 1: Add failing golden tests**

Append to `js/test/golden.test.ts`:

```ts
describe("golden: stacked-bar", () => {
  it("emits a multi-series stacked bar spec", () => {
    const out = assembleFixture("stacked-bar");
    expect(out.type).toBe("bar");
    expect(out.options?.stacked).toBe(true);
    expect(out.data.series.length).toBe(2);
    expect(out.data.series.map((s: any) => s.name).sort()).toEqual(["APAC", "EMEA"]);
    expect(out.data.series[0].color).toMatch(/^#[0-9a-f]{6}$/);
    expectGolden("stacked-bar", out);
  });
});

describe("golden: line-temporal", () => {
  it("emits a timeseries spec with ms X values and a time format", () => {
    const out = assembleFixture("line-temporal");
    expect(out.type).toBe("timeseries");
    const xs = out.data.series[0].values!.map((p: any) => p.x);
    for (const x of xs) expect(typeof x).toBe("number"); // ms since epoch
    expect(xs[0]).toBeGreaterThan(1.7e12);
    expect(out.x_axis?.format?.kind).toBe("time");
    expect(out.x_axis?.format?.layout).not.toMatch(/%/); // fully translated Go layout
    expectGolden("line-temporal", out);
  });
});
```

Run → FAIL (unknown chart types).

- [ ] **Step 2: Implement the templates**

`js/src/ntcharts/templates/stacked-bar.ts` — delegate to the bar template's behavior with the stacked flag forced:

```ts
import type { ChartTemplateDef } from "flint-chart";
import { makeCartesianPivot, makeSortAction } from "flint-chart";
import { ntBarChartDef } from "./bar.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntStackedBarChartDef: ChartTemplateDef = {
  ...ntBarChartDef,
  chart: "Stacked Bar Chart",
  instantiate: (spec: any, rawCtx: any) => {
    ntBarChartDef.instantiate(spec, rawCtx);
    const ctx = rawCtx as NtInstantiateContext;
    ctx.emit.options = { ...ctx.emit.options, stacked: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({
    transpose: [["x", "y"]],
    permute: [["x", "y", "color"]],
    shift: ["color", "group"],
  }),
};
```

`js/src/ntcharts/templates/line.ts`:

```ts
import type { ChartTemplateDef } from "flint-chart";
import { makeCartesianPivot, makeSortAction } from "flint-chart";
import { splitSeries } from "../series.js";
import type { NtInstantiateContext } from "../assemble.js";

// toMs coerces a converted temporal cell (Date | number | ISO string) to
// ms-since-epoch, matching ntcharts-spec's time convention.
function toMs(v: unknown): number {
  if (v instanceof Date) return v.getTime();
  if (typeof v === "number") return v;
  const t = Date.parse(String(v));
  return Number.isNaN(t) ? 0 : t;
}

export const ntLineChartDef: ChartTemplateDef = {
  chart: "Line Chart",
  template: { mark: "line" },
  channels: ["x", "y", "color", "group"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    const temporal = cs.x?.type === "temporal";
    emit.type = temporal ? "timeseries" : "line";
    emit.x_axis = { ...emit.x_axis, type: temporal ? "time" : "value", title: cs.x?.field };
    emit.y_axis = { ...emit.y_axis, title: cs.y?.field };
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;
    if (ctx.ntFormatY) emit.y_axis.format = ctx.ntFormatY;
    if (cs.y?.zero?.zero) emit.y_axis.min = 0;

    const xField = cs.x!.field;
    emit.data.series = splitSeries(table, cs, (row) => ({
      x: temporal ? toMs(row[xField]) : Number(row[xField]),
      y: Number(row[cs.y!.field]),
    }));
    if (emit.data.series.length > 1) emit.options = { ...emit.options, show_legend: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({ permute: [["x", "y", "color"]], shift: ["color", "group"] }),
};
```

Note: `table` rows here come from `overflow.values` which was computed from `convertTemporalData`'s output — verify what that conversion produces for temporal cells (Date objects vs ms numbers vs strings) by inspecting one row while testing, and make `toMs` handle the actual shape (bounded adaptation; record it).

Register both in `templates/index.ts`.

- [ ] **Step 3: Run, inspect goldens, full suite, commit**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npx vitest run && npx tsc --noEmit && npm run build
git -C .. add js/src/ntcharts/templates/ js/test/golden.test.ts testdata/ntspec-golden/stacked-bar.json testdata/ntspec-golden/line-temporal.json
git -C .. commit -m "feat(ts): Stacked Bar + Line/Timeseries templates + goldens"
```

---

### Task 5: Scatter + Heatmap templates (+ additive YAxis.Labels in ntcharts) + goldens

**Files:**
- Create: `js/src/ntcharts/templates/scatter.ts`, `js/src/ntcharts/templates/heatmap.ts`
- Modify: `js/src/ntcharts/templates/index.ts`, `js/test/golden.test.ts`
- Create (generated): `testdata/ntspec-golden/scatter.json`, `testdata/ntspec-golden/heatmap.json`
- Modify (OTHER REPO, /Users/evan/projects/ntcharts branch `spec`): `spec/spec.go`, `spec/spec_test.go`, `spec/README.md` — additive `YAxis.Labels []string` field

**Interfaces:**
- Consumes: everything prior.
- Produces: `"Scatter Plot"` (size channel → `DataPoint.size`) and `"Heatmap"` (categories → cell indices + `x_axis.labels`/`y_axis.labels`, sequential scheme → `theme.gradient`) registered; ntcharts-spec gains `YAxis.Labels` (`json:"labels,omitempty"`, documented as "row labels for grid charts; terminal heatmap currently ignores them" in the fidelity matrix).

- [ ] **Step 1: ntcharts additive change first (other repo, TDD)**

In `/Users/evan/projects/ntcharts` (branch `spec`): add to `spec/spec_test.go`'s round-trip test a `YAxis.Labels: []string{"Mon","Tue"}` assertion → run `go test ./spec/ -run TestSpecJSONRoundTrip` → FAIL → add `Labels []string \`json:"labels,omitempty"\`` to `YAxis` in `spec/spec.go` (after `Max`) → PASS + `go vet ./spec/` clean → add a YAxis.Labels row to README's fidelity matrix (terminal: ignored; web: ignored — future) →

```bash
cd /Users/evan/projects/ntcharts
git add spec/spec.go spec/spec_test.go spec/README.md
git commit -m "feat(spec): additive YAxis.Labels for grid-chart row labels"
```

NEVER `git add -A` here — the four go.mod/go.sum files must stay unstaged.

- [ ] **Step 2: Add failing golden tests**

Append to `js/test/golden.test.ts`:

```ts
describe("golden: scatter", () => {
  it("emits a scatter spec with per-series colors", () => {
    const out = assembleFixture("scatter");
    expect(out.type).toBe("scatter");
    expect(out.data.series.length).toBe(3); // Japan, USA, Germany
    for (const s of out.data.series) expect(s.color).toMatch(/^#[0-9a-f]{6}$/);
    const p = out.data.series[0].values![0];
    expect(typeof p.x).toBe("number");
    expect(typeof p.y).toBe("number");
    expectGolden("scatter", out);
  });
});

describe("golden: heatmap", () => {
  it("emits heat cells with category labels and a gradient", () => {
    const out = assembleFixture("heatmap");
    expect(out.type).toBe("heatmap");
    expect(out.heat?.cells?.length).toBe(9);
    expect(out.x_axis?.labels).toEqual(["09", "12", "15"]);
    expect(out.y_axis?.labels).toEqual(["Mon", "Tue", "Wed"]);
    expect(out.theme?.gradient?.length).toBeGreaterThanOrEqual(5);
    expect(out.data.series).toEqual([]); // heat data lives in heat, not series
    expectGolden("heatmap", out);
  });
});
```

- [ ] **Step 3: Implement scatter**

`js/src/ntcharts/templates/scatter.ts`:

```ts
import type { ChartTemplateDef } from "flint-chart";
import { makeCartesianPivot, makeSortAction } from "flint-chart";
import { splitSeries } from "../series.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntScatterPlotDef: ChartTemplateDef = {
  chart: "Scatter Plot",
  template: { mark: "circle" },
  channels: ["x", "y", "color", "size", "group"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    emit.type = "scatter";
    emit.x_axis = { ...emit.x_axis, type: "value", title: cs.x?.field };
    emit.y_axis = { ...emit.y_axis, title: cs.y?.field };
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;
    if (ctx.ntFormatY) emit.y_axis.format = ctx.ntFormatY;
    if (cs.y?.zero?.zero) emit.y_axis.min = 0;

    const sizeField = cs.size?.field;
    emit.data.series = splitSeries(table, cs, (row) => {
      const p: any = { x: Number(row[cs.x!.field]), y: Number(row[cs.y!.field]) };
      if (sizeField != null) p.size = Number(row[sizeField]);
      return p;
    });
    if (emit.data.series.length > 1) emit.options = { ...emit.options, show_legend: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({
    transpose: [["x", "y"]],
    permute: [["x", "y", "color", "size"]],
    shift: ["color", "group"],
  }),
};
```

- [ ] **Step 4: Implement heatmap**

`js/src/ntcharts/templates/heatmap.ts`:

```ts
import type { ChartTemplateDef } from "flint-chart";
import { detectBandedAxisFromSemantics } from "flint-chart";
import { gradientForScheme } from "../colormap.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntHeatmapDef: ChartTemplateDef = {
  chart: "Heatmap",
  template: { mark: "rect" },
  channels: ["x", "y", "color"],
  markCognitiveChannel: "color",
  declareLayoutMode: (cs, table) => {
    const result = detectBandedAxisFromSemantics(cs, table, { preferAxis: "x" });
    return {
      axisFlags: { x: { banded: true }, y: { banded: true } },
      resolvedTypes: result?.resolvedTypes,
    };
  },
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    emit.type = "heatmap";

    const indexOf = (field: string, preferred?: string[]) => {
      const order: string[] = [];
      const seen = new Set<string>();
      if (preferred) for (const v of preferred) { seen.add(String(v)); order.push(String(v)); }
      for (const row of table) {
        const v = String(row[field]);
        if (!seen.has(v)) { seen.add(v); order.push(v); }
      }
      return order;
    };
    const xLabels = indexOf(cs.x!.field, cs.x!.ordinalSortOrder);
    const yLabels = indexOf(cs.y!.field, cs.y!.ordinalSortOrder);
    const xi = new Map(xLabels.map((l, i) => [l, i]));
    const yi = new Map(yLabels.map((l, i) => [l, i]));

    const zField = cs.color!.field;
    emit.heat = {
      cells: table.map((row) => ({
        x: xi.get(String(row[cs.x!.field]))!,
        y: yi.get(String(row[cs.y!.field]))!,
        z: Number(row[zField]),
      })),
    };
    emit.data.series = [];
    emit.x_axis = { ...emit.x_axis, type: "category", labels: xLabels, title: cs.x?.field };
    emit.y_axis = { ...emit.y_axis, labels: yLabels, title: cs.y?.field };
    const scheme = cs.color?.colorScheme;
    emit.theme = { ...emit.theme, gradient: gradientForScheme(scheme?.scheme) };
  },
};
```

Registry: register scatter + heatmap. NOTE: `emit.data.series = []` conflicts with ntcharts-spec `Validate()` only if heatmaps require series — Phase 2 exempted heatmaps with Heat data, so an empty series array is valid.

- [ ] **Step 5: Run, inspect goldens, full suite, commit**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npx vitest run && npx tsc --noEmit && npm run build
git -C .. add js/src/ntcharts/templates/ js/test/golden.test.ts testdata/ntspec-golden/scatter.json testdata/ntspec-golden/heatmap.json
git -C .. commit -m "feat(ts): Scatter + Heatmap templates + goldens"
```

---

### Task 6: Go cross-validation harness + README + package metadata

**Files:**
- Modify: `go.mod` (flint-ntcharts — add ntcharts dependency + replaces)
- Create: `speccheck/speccheck_test.go`
- Modify: `README.md` (Phase-3 section)
- Modify: `js/package.json` (exports/main pointing at the backend entry for future publish)

**Interfaces:**
- Consumes: goldens from Tasks 3–5; ntcharts `spec` package (branch `spec`, local).
- Produces: `go test ./speccheck/` proving every emitted golden passes `spec.Validate()` AND `spec.Build()` and renders a non-empty terminal view — the end-to-end contract check (flint input → TS backend → JSON → Go render).

- [ ] **Step 1: Wire the Go dependency**

```bash
cd /Users/evan/projects/flint-ntcharts
go mod edit -require=github.com/NimbleMarkets/ntcharts/v2@v2.0.0 \
  -replace=github.com/NimbleMarkets/ntcharts/v2=../ntcharts
```

Then MIRROR ntcharts' own replace directives into this go.mod (a dependency's `replace` lines are ignored, so they must be copied): read `/Users/evan/projects/ntcharts/go.mod`, copy each `replace` line verbatim (there is at least a `charm.land/bubbletea/v2 => github.com/neomantra/...` fork replace), then `go mod tidy`. If tidy fails on the placeholder version, use the version string tidy suggests. Record final go.mod in the report.

- [ ] **Step 2: Write the failing cross-validation test**

`speccheck/speccheck_test.go`:

```go
// Package speccheck cross-validates the TypeScript backend's emitted
// ntcharts-spec JSON against the real Go renderer: every golden must pass
// Validate() and Build() and produce a non-empty terminal view.
package speccheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

func TestGoldensBuildWithNtcharts(t *testing.T) {
	goldens, err := filepath.Glob(filepath.Join("..", "testdata", "ntspec-golden", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(goldens) < 5 {
		t.Fatalf("expected ≥5 goldens, found %d (run the vitest suite first)", len(goldens))
	}
	for _, path := range goldens {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var s spec.Spec
			if err := json.Unmarshal(raw, &s); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if err := s.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			model, err := spec.Build(s)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			view := renderView(t, model)
			if strings.TrimSpace(view) == "" {
				t.Fatal("rendered view is empty")
			}
		})
	}
}

// renderView calls the model's View() string method via a type switch on the
// concrete types spec.Build can return.
func renderView(t *testing.T, model any) string {
	t.Helper()
	type viewer interface{ View() string }
	if v, ok := model.(viewer); ok {
		return v.View()
	}
	t.Fatalf("model %T has no View() string", model)
	return ""
}
```

Run `go test ./speccheck/` → likely FAILS at first (this is the point): JSON-shape mismatches between the TS emitter and the Go structs surface here. Fix mismatches ON THE TS SIDE (types.ts / templates), regenerate goldens (`UPDATE_GOLDEN=1 npx vitest run`), and rerun until green. Record every mismatch found — each one is a contract bug this harness just caught. If a mismatch reveals a genuine ntcharts-spec gap, STOP and report DONE_WITH_CONCERNS rather than changing the Go side unilaterally.

- [ ] **Step 3: README + package metadata**

- `README.md` (flint-ntcharts): add a Phase-3 section — what `assembleNtcharts` is, supported chart types (5 + "Grouped Bar Chart intentionally unsupported"), the cells sizing convention, format translation summary (d3→Go layouts, suffix dropped), how to run vitest/speccheck, golden regeneration.
- `js/package.json`: add `"exports": { ".": "./src/ntcharts/index.ts" }` placeholder note or keep private with a comment — publishing config is a later decision; do not publish.

- [ ] **Step 4: Full verification + commit**

```bash
cd /Users/evan/projects/flint-ntcharts
cd js && npx vitest run && npx tsc --noEmit && npm run build && cd ..
go test ./... 
git add go.mod go.sum speccheck/ README.md js/package.json testdata/ntspec-golden/
git commit -m "feat(go): speccheck cross-validation harness; phase-3 README"
```

Expected: `go test ./...` runs BOTH the Phase-1 parity suite AND speccheck green — the full loop (flint→wasm parity, flint→ntcharts-spec→terminal render) in one command.
