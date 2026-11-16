import { paletteForScheme } from "./colormap.js";
import type { NtInstantiateContext } from "./assemble.js";

type Row = Record<string, unknown>;

/** Rows drawn as one mark (a curve, a band, a set of points), with its name and colour. */
export interface RowGroup { name: string; rows: Row[]; color?: string }

/**
 * The table split into what one terminal plot draws separately: by facet
 * (column, row) and then by a categorical colour. The terminal has one plot,
 * so facets become more groups on it, which the chart says
 * (`facets-flattened`). Colours follow the colour channel's scheme; groups it
 * does not tell apart are told apart by facet.
 */
export function rowGroups(ctx: NtInstantiateContext, colorChannel: "color" | "group" = "color"): RowGroup[] {
  const cs = ctx.channelSemantics;
  const facets = [cs.column, cs.row].filter(Boolean).map((f) => f!.field);
  const colorCS = cs[colorChannel]?.type === "quantitative" ? undefined : cs[colorChannel];
  const facetKeys: string[] = [];
  const colorKeys: string[] = colorCS?.ordinalSortOrder?.map(String) ?? [];
  const buckets = new Map<string, Row[]>();
  for (const row of ctx.table) {
    const f = facets.map((k) => String(row[k])).join(" / ");
    const c = colorCS ? String(row[colorCS.field]) : "";
    if (!facetKeys.includes(f)) facetKeys.push(f);
    if (colorCS && !colorKeys.includes(c)) colorKeys.push(c);
    const key = JSON.stringify([f, c]);
    (buckets.get(key) ?? buckets.set(key, []).get(key)!).push(row);
  }
  if (facets.length) {
    ctx.warn({ severity: "info", code: "facets-flattened",
      message: `the terminal draws one plot: each ${facets.join(" and ")} is drawn on it as its own ${colorCS ? "set of series" : "series"}`,
      channel: "column", field: facets[0] } as any);
  }
  const palette = paletteForScheme(colorCS?.colorScheme?.scheme);
  const groups: RowGroup[] = [];
  for (const f of facetKeys) {
    for (const c of colorCS ? colorKeys : [""]) {
      const rows = buckets.get(JSON.stringify([f, c]));
      if (!rows?.length) continue;
      const name = [f, c].filter(Boolean).join(": ");
      const colorIndex = colorCS ? colorKeys.indexOf(c) : facetKeys.indexOf(f);
      groups.push({ name, rows, ...(colorCS || facets.length ? { color: palette[colorIndex % palette.length] } : {}) });
    }
  }
  // Facets that share a colour need another way to be told apart.
  if (colorCS && facetKeys.length > 1) {
    const all = paletteForScheme(undefined);
    groups.forEach((g, i) => { g.color = all[i % all.length]; });
  }
  return groups;
}
