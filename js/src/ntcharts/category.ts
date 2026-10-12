import { splitSeries } from "./series.js";
import type { NtInstantiateContext } from "./assemble.js";
import type { NtSeries } from "./types.js";

export function isCategorical(ctx: NtInstantiateContext): boolean {
  return ["nominal", "ordinal"].includes(ctx.channelSemantics.x?.type);
}

// Keep labels independently of measure validity: omitting a point must not
// shift later categories. Semantic ordering takes precedence over encounter order.
export function finishCategorical(ctx: NtInstantiateContext, series: NtSeries[], line: boolean): void {
  const field = ctx.channelSemantics.x.field;
  const labels = [...new Set(ctx.table.filter(r => r[field] != null).map(r => String(r[field])))];
  const order = ctx.channelSemantics.x.ordinalSortOrder;
  if (order?.length) {
    const ranks = new Map(order.map((v, i) => [String(v), i]));
    labels.sort((a, b) => (ranks.get(a) ?? Infinity) - (ranks.get(b) ?? Infinity));
  }
  const ranks = new Map(labels.map((v, i) => [v, i]));
  let dropped = 0;
  for (const s of series) {
    const seen = new Set<string>();
    s.values = (s.values ?? []).filter(p => {
      if (p.x == null || !Number.isFinite(p.y)) { dropped++; return false; }
      const key = String(p.x);
      if (line && seen.has(key)) throw new Error(`categorical line series '${s.name}' has duplicate category '${key}'; pre-aggregate its values`);
      seen.add(key);
      return true;
    });
    if (line) s.values.sort((a, b) => ranks.get(String(a.x))! - ranks.get(String(b.x))!);
  }
  ctx.emit.x_axis = { type: "category", title: field, labels };
  ctx.emit.data.series = series;
  if (series.length > 1) ctx.emit.options = { ...ctx.emit.options, show_legend: true };
  if (dropped) ctx.warn({ severity: "warning", code: "invalid-category-point",
    message: `omitted ${dropped} point(s) with missing category or nonfinite measure` });
}

export function categoryPoint(ctx: NtInstantiateContext, row: Record<string, unknown>) {
  const x = row[ctx.channelSemantics.x.field];
  const y = row[ctx.channelSemantics.y.field];
  return { x: x == null ? undefined : String(x), y: y == null || y === "" ? NaN : Number(y) };
}

// Faceted inputs use distinct series in the terminal's single plot. Keeping
// them separate avoids silently connecting unrelated observations.
export function categorySeries(ctx: NtInstantiateContext): NtSeries[] {
  const facets = [ctx.channelSemantics.column, ctx.channelSemantics.row].filter(Boolean);
  const semantics = { ...ctx.channelSemantics, group: ctx.channelSemantics.group ?? ctx.channelSemantics.detail };
  const point = (row: Record<string, unknown>) => ({ ...categoryPoint(ctx, row),
    ...(ctx.channelSemantics.size ? { size: Number(row[ctx.channelSemantics.size.field]) } : {}) });
  if (!facets.length) return splitSeries(ctx.table, semantics, point);
  const buckets = new Map<string, Record<string, unknown>[]>();
  for (const row of ctx.table) {
    const key = JSON.stringify(facets.map(cs => row[cs.field]));
    (buckets.get(key) ?? buckets.set(key, []).get(key)!).push(row);
  }
  ctx.warn({severity: "info", code: "facets-flattened",
    message: "terminal categorical chart draws facets as separate series on one plot"});
  return [...buckets].flatMap(([key, rows]) => splitSeries(rows, semantics, point)
    .map(s => ({ ...s, name: `${JSON.parse(key).map(String).join(" / ")}: ${s.name}` })));
}
