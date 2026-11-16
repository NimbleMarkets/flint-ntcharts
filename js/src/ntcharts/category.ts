import { splitSeries } from "./series.js";
import { paletteForScheme } from "./colormap.js";
import { bandTicks } from "./bands.js";
import { centerSlots } from "./ticks.js";

// Columns the Y axis labels and line take from a line chart's width.
export const Y_LABEL_ROOM = 9;
import type { NtInstantiateContext } from "./assemble.js";
import type { NtSeries } from "./types.js";

const discrete = (t: string | undefined) => t === "nominal" || t === "ordinal";

/**
 * The axes as a terminal plot can draw them. ntcharts measures along y only: a
 * categorical y beside a measured x is drawn transposed (the categories along
 * x), and said so; two categorical axes cannot be drawn at all.
 */
export function orientMeasure(ctx: NtInstantiateContext): NtInstantiateContext {
  const cs = ctx.channelSemantics;
  const shown = () => (["x", "y"] as const).map((a) => cs[a] ? `${a} = ${cs[a].field} (${cs[a].type})` : `${a} = none`).join(", ");
  if (!cs.x || !cs.y) throw new Error(`A terminal ${ctx.chartType} needs both x and y; got ${shown()}.`);
  if (discrete(cs.x.type) && discrete(cs.y.type)) {
    throw new Error(`A terminal ${ctx.chartType} needs a measured axis; got ${shown()}. Use a Heatmap for two categorical axes.`);
  }
  if (cs.y.type === "quantitative") return ctx;
  ctx.warn({ severity: "info", code: "axes-transposed",
    message: `drawn transposed: ${cs.y.field} along x and ${cs.x.field} up y, as the terminal measures along y only`,
    channel: "y", field: cs.y.field } as any);
  return {
    ...ctx,
    channelSemantics: { ...cs, x: cs.y, y: cs.x },
    encodings: { ...ctx.encodings, x: ctx.encodings.y, y: ctx.encodings.x },
    ntFormatX: ctx.ntFormatY,
    ntFormatY: ctx.ntFormatX,
  };
}

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
  const cs = ctx.channelSemantics.x;
  const ordered = cs.type === "ordinal" || !!cs.ordinalSortOrder?.length;
  const ticks = bandTicks(ctx, labels, labels, ordered, centerSlots(labels.length, Math.max(1, ctx.emit.width - Y_LABEL_ROOM)), field, "x");
  if (ticks) ctx.emit.x_axis.tick_labels = ticks;
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
  const flat = [...buckets].flatMap(([key, rows]) => splitSeries(rows, semantics, point)
    // A facet's series is named by the facet, and by its colour where it has one.
    .map(s => {
      const facet = JSON.parse(key).map(String).join(" / ");
      return { ...s, name: s.name === ctx.channelSemantics.y?.field ? facet : `${facet}: ${s.name}` };
    }));
  // Series that the colour channel did not tell apart are told apart by facet.
  const palette = paletteForScheme(undefined);
  return flat.map((s, i) => (flat.length > 1 && !s.color ? { ...s, color: palette[i % palette.length] } : s));
}
