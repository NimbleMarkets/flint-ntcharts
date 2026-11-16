import type { ChartTemplateDef } from "flint-chart/core";
import { detectBandedAxisForceDiscrete, makeCartesianPivot, makeSortAction, planBandDodge, resolveDodge } from "flint-chart/core";
import { ntBarChartDef } from "./bar.js";
import { paletteForScheme } from "../colormap.js";
import { bandTicks, categoryKey, categoryLabels, categoryOrder, combine, combineOp, wantValueLabels } from "../bands.js";
import { barSlots } from "../ticks.js";
import type { NtInstantiateContext } from "../assemble.js";

type Row = Record<string, unknown>;

const groupFieldOf = (cs: Record<string, any>) => cs.group?.field ?? cs.color?.field;

// The most bars a category can set side by side and still tell apart: one per
// colour of the categorical palette.
const MAX_LANES = 10;

function dodgeOf(cs: Record<string, any>, table: Row[], axis: "x" | "y", override: unknown) {
  const groupField = groupFieldOf(cs);
  const axisField = cs[axis]?.field;
  if (!groupField || !axisField || groupField === axisField) return { mode: "none" as const, laneCount: 1 };
  return resolveDodge(planBandDodge(table, axisField, groupField), override as string | undefined);
}

// ntcharts draws only stacked bars, so grouped bars are laid out as one bar
// per group, each a single coloured segment, with an empty bar between
// categories. Lanes follow flint's dodge plan: `global` gives every category
// a lane for every group (missing ones stay empty, so lanes align), `local`
// only the groups it has. A category's name sits under its first bar.
export const ntGroupedBarChartDef: ChartTemplateDef = {
  chart: "Grouped Bar Chart",
  template: { mark: "bar" },
  channels: ["x", "y", "group", "color", "column", "row"],
  markCognitiveChannel: "length",
  // The grouping goes on the group channel, so flint budgets a lane for each
  // group; a grouping with more values than the bars can tell apart is dropped
  // and each bar combines them.
  normalizeEncodings: ((encodings: Record<string, any>, data: Row[], warn?: (w: any) => void) => {
    const grouping = encodings.group ?? encodings.color;
    if (!grouping?.field) return encodings;
    // Grouped by its own category: nothing to set side by side, only colour.
    if (grouping.field === encodings.x?.field || grouping.field === encodings.y?.field) {
      const { group: _g, ...rest } = encodings;
      return { ...rest, color: grouping };
    }
    const count = new Set(data.map((r) => r[grouping.field])).size;
    if (count > MAX_LANES) {
      warn?.({ severity: "warning", code: "too-many-groups",
        message: `${grouping.field} has ${count} values, too many to set side by side: each bar combines them`,
        channel: "group", field: grouping.field });
      const { group: _g, color: _c, ...rest } = encodings;
      return rest;
    }
    return encodings.group ? encodings : { ...encodings, group: encodings.color };
  }) as any,
  declareLayoutMode: (cs: any, table: any, chartProperties?: any) => {
    const result = detectBandedAxisForceDiscrete(cs, table, { preferAxis: "x" });
    const axis = (result?.axis ?? "x") as "x" | "y";
    const { mode, laneCount } = dodgeOf(cs, table, axis, chartProperties?.dodge);
    // Two cells per bar (the bar and its gap). Flint budgets a grouped
    // category max(3 / stepPadding, 2 × groups) cells; this stepPadding makes
    // that room for the lanes and one empty bar between categories.
    const paramOverrides = mode === "none" ? { minStep: 2 } : { minStep: 2, stepPadding: 3 / (2 * (laneCount + 1)) };
    return { axisFlags: { [axis]: { banded: true } }, resolvedTypes: result?.resolvedTypes, paramOverrides };
  },
  instantiate: (spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table, encodings } = ctx;
    const horizontal = cs.x?.type === "quantitative" && cs.y != null && cs.y.type !== "quantitative";
    const catAxis = horizontal ? "y" : "x";
    const valAxis = horizontal ? "x" : "y";
    const catCS = cs[catAxis];
    const valCS = cs[valAxis];
    const groupCS = cs.group ?? cs.color;
    const { mode } = dodgeOf(cs, table, catAxis, ctx.chartProperties?.dodge);
    if (!groupCS || mode === "none" || !catCS || valCS?.type !== "quantitative") {
      // Nothing to set side by side: the bars share each band, as a bar chart's do.
      ntBarChartDef.instantiate(spec, rawCtx);
      return;
    }

    const op = combineOp(valCS, encodings[valAxis]?.aggregate);
    const measure = (rows: Row[]) => {
      const vs = rows.map((r) => Number(r[valCS.field])).filter(Number.isFinite);
      return vs.length ? combine(vs, op) : 0;
    };
    const keys = categoryOrder(table, catCS, encodings[catAxis], measure);
    const groups = [...new Set(table.map((r) => String(r[groupCS.field])))];
    const preferred = groupCS.ordinalSortOrder?.map(String);
    if (preferred) groups.sort((a, b) => (preferred.indexOf(a) + 1 || Infinity) - (preferred.indexOf(b) + 1 || Infinity));
    else if (groupCS.type === "quantitative") groups.sort((a, b) => Number(a) - Number(b));

    const cells = new Map<string, Row[]>();
    let combined = false;
    for (const row of table) {
      if (row[catCS.field] == null) continue;
      const key = JSON.stringify([categoryKey(row[catCS.field]), String(row[groupCS.field])]);
      const rows = cells.get(key) ?? cells.set(key, []).get(key)!;
      rows.push(row);
      if (rows.length > 1) combined = true;
    }

    const lanesOf = (k: string) => (mode === "global" ? groups : groups.filter((g) => cells.has(JSON.stringify([k, g]))));
    // Each bar spends two cells; categories that would leave a bar none are cut.
    const room = Math.floor(((horizontal ? emit.height : emit.width) + 1) / 2);
    let used = -1;
    const fit = keys.findIndex((k) => (used += lanesOf(k).length + 1) > room);
    if (fit > 0) {
      ctx.warn({ severity: "warning", code: "overflow",
        message: `${keys.length - fit} of ${keys.length} values in '${catCS.field}' were omitted (showing first ${fit} in sort order).`,
        channel: catAxis, field: catCS.field } as any);
      keys.length = fit;
    }
    const names = categoryLabels(keys);
    const bars: { label: string; group?: string; value: number }[] = [];
    keys.forEach((k, i) => {
      const lanes = lanesOf(k);
      if (i > 0) bars.push({ label: "", value: 0 });
      lanes.forEach((g, j) => bars.push({ label: j === 0 ? names[i] : "", group: g, value: measure(cells.get(JSON.stringify([k, g])) ?? []) }));
    });

    const labels = bars.map((b) => b.label);
    // Each category is named under its first bar; plan those names over the
    // columns where the categories start.
    let tickLabels: string[] | undefined;
    if (!horizontal) {
      const firsts = bars.map((b, i) => (b.label ? i : -1)).filter((i) => i >= 0);
      const { cols } = barSlots(bars.length, emit.width);
      const ordered = catCS.type !== "nominal" && !encodings[catAxis]?.sortBy;
      const ticks = bandTicks(ctx, keys, names, ordered, { cols: firsts.map((b) => cols[b]), width: emit.width, anchor: "start" }, catCS.field, catAxis);
      if (ticks) {
        tickLabels = labels.map(() => "");
        firsts.forEach((b, i) => { tickLabels![b] = ticks[i]; });
      }
    }

    const palette = paletteForScheme(groupCS.type === "quantitative" ? undefined : groupCS.colorScheme?.scheme);
    emit.type = "bar";
    emit.options = { ...emit.options, stacked: true, show_legend: true, ...(horizontal ? { orientation: "horizontal" } : {}) };
    emit.x_axis = { ...emit.x_axis, type: "category", labels, title: catCS.field, ...(tickLabels ? { tick_labels: tickLabels } : {}) };
    emit.y_axis = { ...emit.y_axis, title: valCS.field };
    const valueFormat = horizontal ? ctx.ntFormatX : ctx.ntFormatY;
    if (valueFormat) emit.y_axis.format = valueFormat;
    emit.data.series = groups.map((g, i) => ({
      name: g,
      values: bars.map((b) => ({ y: b.group === g ? b.value : 0 })),
      color: palette[i % palette.length],
    }));
    if (valCS.zero?.zero && bars.every((b) => b.value >= 0)) emit.y_axis.min = 0;
    const values = bars.filter((b) => b.group).map((b) => b.value);
    if (wantValueLabels(ctx.chartProperties, values, emit.y_axis.format, Math.floor(emit.width * values.length / bars.length), horizontal)) {
      emit.options = { ...emit.options, show_values: true };
    }

    const facets = [cs.column, cs.row].filter(Boolean).map((f) => f!.field);
    if (facets.length) {
      ctx.warn({ severity: "info", code: "facets-merged",
        message: `the terminal draws one plot: each bar ${op === "average" ? "averages" : "combines"} its rows across ${facets.join(" and ")} (${op})`,
        channel: "column", field: facets[0] } as any);
    } else if (combined && !encodings[valAxis]?.aggregate) {
      ctx.warn({ severity: "info", code: "rows-combined",
        message: `several rows share a ${catCS.field} and ${groupCS.field}: each bar shows their ${op}`,
        channel: valAxis, field: valCS.field } as any);
    }
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({ transpose: [["x", "y"]], permute: [["x", "y", "group"]], shift: ["group", "color"] }),
};
