import type { ChannelSemantics } from "flint-chart/core";
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
  // Assign color by position in the full `order` list BEFORE filtering out
  // empty buckets: an empty preferred category (e.g. a canonical
  // ordinalSortOrder entry with zero rows in this dataset) must not shift
  // the palette index of series that come after it. This keeps a category's
  // color stable across datasets whenever flint resolves an
  // ordinalSortOrder (the common case for repeated datasets of the same
  // shape). Without an ordinalSortOrder, `order` falls back to
  // first-appearance in THIS dataset, so a category's color can still shift
  // if the appearance order of the surviving categories changes between
  // datasets (e.g. one dataset's first row differs from another's).
  return order
    .map((name, i) => ({ name, values: buckets.get(name) ?? [], color: palette[i % palette.length] }))
    .filter((s) => s.values.length > 0);
}
