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
