import { resolvePivotAggregate } from "flint-chart/core";
import type { ChannelSemantics } from "flint-chart/core";
import { barSlots, planTicks, tickKind, type Slots } from "./ticks.js";
import { valueText } from "./format.js";
import type { NtFormat } from "./types.js";

// Shared reading of a banded (category) axis for the bar family: which
// categories it shows, in what order, under what names, and how several rows
// on one category combine. Terminal bars are always bands, as Flint's Chart.js
// bars are, so this follows that backend: ordinal order, then a sort-by-measure
// request, else first appearance; dates in time order.

type Row = Record<string, unknown>;

/** A category's key: its value as written, so a label and a row agree. */
export const categoryKey = (value: unknown): string => String(value);

const isDateString = (s: string) => /^\d{4}-\d{2}(-\d{2})?([T ]\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:?\d{2})?)?$/.test(s);

/**
 * The categories along a banded axis, in display order. A semantic order
 * (months, weekdays) comes first; dates run in time order; a sort-by-measure
 * request (`sortBy`, `sortOrder`) orders by each category's combined measure;
 * otherwise the order the rows give.
 */
export function categoryOrder(
  table: Row[],
  catCS: ChannelSemantics,
  encoding: { sortBy?: string; sortOrder?: "ascending" | "descending" } | undefined,
  measure: (rows: Row[]) => number,
): string[] {
  const field = catCS.field;
  const seen: string[] = [];
  const rowsOf = new Map<string, Row[]>();
  for (const row of table) {
    if (row[field] == null) continue;
    const key = categoryKey(row[field]);
    if (!rowsOf.has(key)) { rowsOf.set(key, []); seen.push(key); }
    rowsOf.get(key)!.push(row);
  }
  let order = seen;
  const preferred = catCS.ordinalSortOrder;
  if (preferred?.length) {
    const rank = new Map(preferred.map((v, i) => [categoryKey(v), i]));
    order = [...seen].sort((a, b) => (rank.get(a) ?? Infinity) - (rank.get(b) ?? Infinity));
  } else if (seen.length > 1 && seen.every(isDateString)) {
    order = [...seen].sort((a, b) => Date.parse(a) - Date.parse(b));
  }
  if (encoding?.sortBy && encoding.sortOrder) {
    const dir = encoding.sortOrder === "ascending" ? 1 : -1;
    const value = new Map(order.map((k) => [k, measure(rowsOf.get(k) ?? [])]));
    order = [...order].sort((a, b) => dir * ((value.get(a) ?? 0) - (value.get(b) ?? 0)));
  }
  return order;
}

/**
 * How several rows on one category combine into the one bar the terminal
 * draws: the encoding's own `aggregate`, else the measure's semantic default
 * (sum for additive measures, average for intensive ones and for `Number`,
 * whose additivity is unknown). Flint 1.1's rule for exporters that pivot raw
 * rows (`resolvePivotAggregate`).
 */
export function combineOp(valCS: ChannelSemantics, aggregate: string | undefined): "sum" | "average" | "count" {
  return resolvePivotAggregate(aggregate, valCS.aggregationDefault);
}

export function combine(values: number[], op: "sum" | "average" | "count"): number {
  if (op === "count") return values.length;
  const total = values.reduce((a, b) => a + b, 0);
  return op === "average" ? total / values.length : total;
}

/**
 * The name a date category is drawn under: as short as its values allow and
 * still tell them apart — the year where every date is a year's first day, the
 * month where every date is a month's first day, else the day. Times of day
 * keep their minutes. A value that is not a date is drawn as written.
 */
export function categoryLabels(keys: string[]): string[] {
  if (!keys.length || !keys.every(isDateString)) return keys;
  const parts = keys.map((k) => ({ k, date: k.slice(0, 10), time: k.length > 10 ? k.slice(11, 16) : "" }));
  if (parts.some((p) => p.time && p.time !== "00:00")) return keys.map((k) => k.slice(0, 16).replace("T", " "));
  const unique = (labels: string[]) => new Set(labels).size === labels.length;
  const day = (d: string) => (d.length === 7 ? "01" : d.slice(8, 10));
  const firstOfMonth = parts.every((p) => day(p.date) === "01");
  const years = parts.map((p) => p.date.slice(0, 4));
  if (firstOfMonth && parts.every((p) => p.date.slice(5, 7) === "01") && unique(years)) return years;
  const months = parts.map((p) => p.date.slice(0, 7));
  if (firstOfMonth && unique(months)) return months;
  return parts.map((p) => p.date);
}

/**
 * The names drawn under a band axis (`x_axis.tick_labels`), planned to fit
 * `slots`: thinned on an ordered axis, cut with an ellipsis on a nominal one
 * (with a `labels-shortened` note). Undefined when every name fits whole.
 */
export function bandTicks(
  ctx: { warn: (w: any) => void; chartProperties?: Record<string, unknown> },
  keys: string[], display: string[], ordered: boolean, slots: Slots, field: string, channel: string,
): string[] | undefined {
  const stride = Number(ctx.chartProperties?.labelStride);
  const kind = tickKind(keys, ordered);
  const plan = planTicks(keys, display, kind, slots, Number.isFinite(stride) && stride >= 1 ? stride : undefined);
  if (plan.shortened) {
    ctx.warn({ severity: "info", code: "labels-shortened",
      message: `names of ${field} cut to fit: the axis runs ${display[0]} to ${display[display.length - 1]}`, channel, field });
  }
  return plan.ticks.every((t, i) => t === display[i]) ? undefined : plan.ticks;
}

/**
 * Whether bars print their values: where every value fits at its bar's end
 * and there are few enough bars to read them (chartProperties.valueLabels
 * overrides). Terminal bars have no value axis, so this is how they say
 * what they measure.
 */
export function wantValueLabels(
  props: Record<string, unknown> | undefined, totals: number[], format: NtFormat | undefined, width: number, horizontal: boolean,
): boolean {
  if (typeof props?.valueLabels === "boolean") return props.valueLabels;
  const widest = Math.max(0, ...totals.map((t) => valueText(t, format).length));
  // A vertical bar's value prints within the bar's own width.
  const room = horizontal ? width : (barSlots(totals.length, width).cols[1] ?? width + 1) - 1;
  return totals.length <= 16 && widest <= room;
}
