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
