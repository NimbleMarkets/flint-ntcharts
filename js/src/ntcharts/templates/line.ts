import { isCategorical, finishCategorical, categorySeries, Y_LABEL_ROOM } from "../category.js";
import type { ChartTemplateDef } from "flint-chart/core";
import { makeCartesianPivot, makeSortAction } from "flint-chart/core";
import { splitSeries } from "../series.js";
import { toMs } from "../temporal.js";
import type { NtInstantiateContext } from "../assemble.js";
import { pinFittedYDomain, pinZeroBaseline } from "../domain.js";
import { regularDates } from "../cadence.js";

export const ntLineChartDef: ChartTemplateDef = {
  chart: "Line Chart",
  template: { mark: "line" },
  channels: ["x", "y", "color", "group"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    let ctx = rawCtx as NtInstantiateContext;
    // Dates at a regular cadence, with at least two columns each, are drawn
    // on category slots, each point over its date; the axis names them in
    // the fewest characters that fit (see ticks.ts). ntcharts' time axis
    // instead labels a fixed 4-column step, a few columns off each date.
    const x0 = ctx.channelSemantics.x;
    if (x0?.type === "temporal") {
      const values = ctx.table.map((r) => r[x0.field]);
      const dates = values.every((v) => v == null || Number.isFinite(toMs(v))) ? regularDates(values.filter((v) => v != null)) : undefined;
      if (dates && dates.length <= Math.floor((ctx.emit.width - Y_LABEL_ROOM) / 2)) {
        const key = (v: unknown) => {
          const iso = new Date(toMs(v)).toISOString();
          return iso.endsWith("T00:00:00.000Z") ? iso.slice(0, 10) : iso.slice(0, 16);
        };
        const table = ctx.table.map((r) => (r[x0.field] == null ? r : { ...r, [x0.field]: key(r[x0.field]) }));
        ctx = { ...ctx, table, channelSemantics: { ...ctx.channelSemantics, x: { ...x0, type: "ordinal", ordinalSortOrder: dates.map(key) } } };
      }
    }
    const { emit, channelSemantics: cs, table } = ctx;
    if (cs.y?.type !== "quantitative") {
      throw new Error(`A terminal ${ctx.chartType} measures along y; got y = ${cs.y ? `${cs.y.field} (${cs.y.type})` : "none"}. ` +
        `Put the measure on y and the ${cs.y?.field ?? "category"} on x.`);
    }
    const temporal = cs.x?.type === "temporal";
    emit.type = temporal ? "timeseries" : "line";
    emit.x_axis = { ...emit.x_axis, type: temporal ? "time" : "value", title: cs.x?.field };
    emit.y_axis = { ...emit.y_axis, title: cs.y?.field };
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;
    if (ctx.ntFormatY) emit.y_axis.format = ctx.ntFormatY;
    if (cs.y?.zero?.zero) pinZeroBaseline(emit, cs, table);
    else pinFittedYDomain(emit, cs, table);

    if (isCategorical(ctx)) {
      finishCategorical(ctx, categorySeries(ctx), true);
      return;
    }

    const xField = cs.x!.field;
    emit.data.series = splitSeries(table, cs, (row) => ({
      x: temporal ? toMs(row[xField]) : Number(row[xField]),
      y: Number(row[cs.y!.field]),
    }));

    // ntcharts' timeserieslinechart/linechart assume chronological push
    // order (pure append, segments drawn by insertion index), but our
    // emitted series preserve raw row order from the source data. Sort each
    // series ascending by x so the drawn segments are correct regardless of
    // input ordering. Also drop any points whose x or y failed to parse
    // (e.g. an unparsable temporal cell yields NaN from toMs) rather than
    // silently plotting a fake epoch-0/NaN point; surface a single warning
    // if anything was dropped.
    let droppedCount = 0;
    for (const series of emit.data.series) {
      const values = series.values ?? [];
      const kept = values.filter((p) => {
        const ok = !Number.isNaN(p.x as number) && !Number.isNaN(p.y as number);
        if (!ok) droppedCount++;
        return ok;
      });
      kept.sort((a, b) => (a.x as number) - (b.x as number));
      series.values = kept;
    }
    if (droppedCount > 0) {
      ctx.warn({
        severity: "warning",
        code: "invalid-temporal-x",
        message: `dropped ${droppedCount} point(s) with unparsable X values`,
      });
    }

    if (emit.data.series.length > 1) emit.options = { ...emit.options, show_legend: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({ permute: [["x", "y", "color"]], shift: ["color", "group"] }),
};
