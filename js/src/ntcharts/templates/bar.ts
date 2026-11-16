import type { ChartTemplateDef } from "flint-chart/core";
import { detectBandedAxisForceDiscrete, makeCartesianPivot, makeSortAction } from "flint-chart/core";
import { paletteForScheme } from "../colormap.js";
import { bandTicks, categoryKey, categoryLabels, categoryOrder, combine, combineOp, wantValueLabels } from "../bands.js";
import { barSlots, ellipsize, tickKind } from "../ticks.js";
import type { NtInstantiateContext } from "../assemble.js";

type Row = Record<string, unknown>;

/** Whether every name stands whole under its bar. */
function fitsUnder(labels: string[], width: number): boolean {
  const { cols } = barSlots(labels.length, width);
  const room = (cols[1] ?? width + 1) - 1;
  return labels.every((l) => [...l].length <= room);
}

export const ntBarChartDef: ChartTemplateDef = {
  chart: "Bar Chart",
  template: { mark: "bar" },
  channels: ["x", "y", "color", "group"],
  markCognitiveChannel: "length",
  declareLayoutMode: (cs, table) => {
    // Terminal bars are always bands, as Flint's Chart.js bars are: a date or
    // number axis is laid out as categories, so flint budgets and truncates it
    // (with its usual overflow warning) rather than handing the renderer more
    // bars than it has cells for.
    const result = detectBandedAxisForceDiscrete(cs, table, { preferAxis: "x" });
    return {
      axisFlags: result ? { [result.axis]: { banded: true } } : { x: { banded: true } },
      resolvedTypes: result?.resolvedTypes,
      // The ntcharts bar model spends two cells per category — the bar and
      // the gap after it — and draws nothing at all once the bar width
      // reaches zero. A 2-cell step makes flint budget to what it can draw.
      paramOverrides: { minStep: 2 },
    };
  },
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table, encodings } = ctx;
    // The measured axis is the quantitative one; with both quantitative, y.
    const horizontal = cs.x?.type === "quantitative" && cs.y != null && cs.y.type !== "quantitative";
    const catAxis = horizontal ? "y" : "x";
    const valAxis = horizontal ? "x" : "y";
    const catCS = cs[catAxis];
    const valCS = cs[valAxis];
    if (!catCS || !valCS || valCS.type !== "quantitative") {
      const shown = (["x", "y"] as const).map((a) => cs[a] ? `${a} = ${cs[a].field} (${cs[a].type})` : `${a} = none`).join(", ");
      throw new Error(`A terminal ${ctx.chartType} needs one quantitative axis to measure; got ${shown}. ` +
        `Use a Heatmap for two categorical axes.`);
    }

    // A facet becomes one more dimension the single terminal plot cannot show:
    // its rows are combined into each bar, and the chart says so.
    const facets = [cs.column, cs.row].filter(Boolean).map((f) => f!.field);
    const groupCS = cs.color?.type === "quantitative" ? undefined : (cs.color ?? cs.group);
    const op = combineOp(valCS, encodings[valAxis]?.aggregate);
    const values = (rows: Row[]) => rows.map((r) => Number(r[valCS.field])).filter(Number.isFinite);
    const measure = (rows: Row[]) => (rows.length ? combine(values(rows), op) : 0);

    emit.type = "bar";
    let keys = categoryOrder(table, catCS, encodings[catAxis], measure);
    let labels = categoryLabels(keys);
    emit.x_axis = { ...emit.x_axis, type: "category", labels, title: catCS.field };
    // The category axis reads in its natural order unless sorted by the measure.
    const ordered = catCS.type !== "nominal" && !encodings[catAxis]?.sortBy;
    const asked = ctx.chartProperties?.orientation;
    let across = asked === "horizontal" || (asked !== "vertical" && horizontal);
    if (!across && !asked && tickKind(keys, ordered) === "nominal" && !fitsUnder(labels, emit.width) && keys.length * 2 - 1 <= emit.height) {
      // Names too long to stand under their bars read whole beside them.
      across = true;
      ctx.warn({ severity: "info", code: "bars-horizontal",
        message: `drawn as horizontal bars so each ${catCS.field} name fits`, channel: catAxis, field: catCS.field } as any);
    }
    if (across) {
      emit.options = { ...emit.options, orientation: "horizontal" };
      // A horizontal bar takes a row and the gap after it.
      const room = Math.max(1, Math.floor((emit.height + 1) / 2));
      if (keys.length > room) {
        ctx.warn({ severity: "warning", code: "overflow",
          message: `${keys.length - room} of ${keys.length} values in '${catCS.field}' were omitted (showing first ${room} in sort order).`,
          channel: catAxis, field: catCS.field } as any);
        keys = keys.slice(0, room);
        labels = labels.slice(0, room);
        emit.x_axis.labels = labels;
      }
      const cap = Math.max(4, Math.floor(emit.width / 3));
      if (labels.some((l) => [...l].length > cap)) emit.x_axis.tick_labels = labels.map((l) => ellipsize(l, cap));
    } else {
      const ticks = bandTicks(ctx, keys, labels, ordered, barSlots(keys.length, emit.width), catCS.field, catAxis);
      if (ticks) emit.x_axis.tick_labels = ticks;
    }
    emit.y_axis = { ...emit.y_axis, title: valCS.field };
    const valueFormat = horizontal ? ctx.ntFormatX : ctx.ntFormatY;
    if (valueFormat) emit.y_axis.format = valueFormat;

    // One series per colour (or group) value, each aligned to the category
    // order; several rows on one bar combine by the measure's rule.
    const seriesKeys: string[] = [];
    const cells = new Map<string, Map<string, Row[]>>();
    const preferred = groupCS?.ordinalSortOrder?.map(String);
    if (preferred) for (const g of preferred) { seriesKeys.push(g); cells.set(g, new Map()); }
    for (const row of table) {
      if (row[catCS.field] == null) continue;
      const g = groupCS ? String(row[groupCS.field]) : valCS.field;
      if (!cells.has(g)) { seriesKeys.push(g); cells.set(g, new Map()); }
      const k = categoryKey(row[catCS.field]);
      const bucket = cells.get(g)!;
      (bucket.get(k) ?? bucket.set(k, []).get(k)!).push(row);
    }
    const palette = paletteForScheme(groupCS?.colorScheme?.scheme);
    let combined = false;
    const series = seriesKeys.map((name, i) => {
      const bucket = cells.get(name)!;
      for (const rows of bucket.values()) if (rows.length > 1) combined = true;
      return {
        name,
        values: keys.map((k) => ({ y: measure(bucket.get(k) ?? []) })),
        ...(groupCS ? { color: palette[i % palette.length] } : {}),
      };
    }).filter((s) => cells.get(s.name)!.size > 0);
    emit.data.series = series;
    if (series.length > 1) emit.options = { ...emit.options, stacked: true, show_legend: true };

    // Terminal bars have no value axis, so each bar says its value where the
    // value fits at its end (chartProperties.valueLabels overrides).
    const totals = keys.map((_, i) => series.reduce((sum, s) => sum + s.values[i].y, 0));
    if (wantValueLabels(ctx.chartProperties, totals, emit.y_axis?.format, emit.width, across)) emit.options = { ...emit.options, show_values: true };

    // The bar model always draws from zero; pinning zero as the minimum would
    // only hide bars that go below it.
    const lowest = Math.min(0, ...series.flatMap((s) => s.values.map((p) => p.y)));
    if (valCS.zero?.zero && lowest >= 0) emit.y_axis.min = 0;

    if (facets.length) {
      ctx.warn({ severity: "info", code: "facets-merged",
        message: `the terminal draws one plot: each bar ${op === "average" ? "averages" : "combines"} its rows across ${facets.join(" and ")} (${op})`,
        channel: "column", field: facets[0] } as any);
    } else if (combined && !encodings[valAxis]?.aggregate) {
      ctx.warn({ severity: "info", code: "rows-combined",
        message: `several rows share a ${catCS.field}${groupCS ? ` and ${groupCS.field}` : ""}: each bar shows their ${op}`,
        channel: valAxis, field: valCS.field } as any);
    }
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({
    transpose: [["x", "y"]],
    permute: [["x", "y", "color"]],
    shift: ["color", "group"],
  }),
};
