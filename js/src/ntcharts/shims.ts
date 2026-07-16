// Local stand-ins for flint-chart core utilities that exist upstream but are
// not exported from the npm package (applyAggregation, resolveBaseSize,
// deriveStretchCaps). Semantics documented from the upstream source; if an
// upstream release exports them, delete this module and import instead.
//
// Deviations (intentionally not shimmed / not implemented here):
//   - normalizeChartProperties is NOT shimmed. `chart_spec.chartProperties`
//     passes through this backend unvalidated: property-driven overrides
//     that upstream applies from normalized chartProperties (e.g.
//     includeZero_x, logScale_x) are NOT applied by assembleNtcharts.
//   - computeMinSubplotDimensions is out of scope (facet layout is not
//     supported by this terminal backend) and intentionally unimplemented.
//   - decideColorMaps is intentionally unimplemented; superseded here by
//     this package's own colormap.ts (gradientForScheme / palette
//     selection), which serves the terminal ntcharts-spec color needs.
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
const averageFn = (v: number[]) => v.reduce((a, b) => a + b, 0) / v.length;
const AGGS: Record<string, (vals: number[]) => number> = {
  sum: (v) => v.reduce((a, b) => a + b, 0),
  average: averageFn,
  mean: averageFn,
  count: (v) => v.length,
  min: (v) => Math.min(...v),
  max: (v) => Math.max(...v),
  median: (v) => { const s = [...v].sort((a, b) => a - b); const m = s.length >> 1;
    return s.length % 2 ? s[m] : (s[m - 1] + s[m]) / 2; },
};

// applyAggregationShim mirrors upstream core/aggregate.ts behavior: when any
// encoding declares `aggregate`, group rows by every OTHER encoded field and
// collapse each aggregate-encoded field with its function. Upstream ALSO
// builds a fresh output row per group (`out.push(aggregated)` over a new
// `{}`) rather than mutating the original row array, so that is not a
// divergence — the real differences are:
//
//   - Derived-column contract: upstream writes the aggregated value into a
//     new column named `${field}_${op}` (`_count` for the `count` op) AND
//     keeps the original `field` populated with the same value (for
//     non-count ops), so downstream assemblers can reference either name.
//     This shim writes only into the original `field`, with no `_op`-suffixed
//     column. Upstream also short-circuits to a no-op when every derived
//     column is already present in the first row (caller pre-aggregated);
//     this shim has no such short-circuit and always re-groups/re-reduces.
//   - Supported ops: this shim implements real `min`/`max`/`median`
//     reducers (see `AGGS` above). Upstream has no `min`/`max`/`median`
//     support at all — its `reduceOp` only handles `count`, `sum`, and the
//     `average`/`mean` synonym pair (arithmetic mean); any other `op` string
//     falls through its sum/mean branch rather than computing a min/max/
//     median. A caller relying on this shim's min/max/median has no upstream
//     equivalent to fall back to.
//
// This shim also has no special empty-bucket handling — every group here is
// built from at least one matching row (groups come from `Map` entries
// populated while iterating `rows`), so an empty bucket cannot occur;
// upstream's behavior for a genuinely empty aggregate bucket (e.g. a
// declared category with zero matching rows) is unverified against this
// implementation.
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
      const op = (enc as any).aggregate as string;
      if (op === "count") {
        merged[field] = bucket.length;
      } else {
        const fn = AGGS[op] ?? AGGS.sum;
        merged[field] = fn(bucket.map((r) => Number(r[field])).filter((n) => !Number.isNaN(n)));
      }
    }
    out.push(merged);
  }
  return out;
}
