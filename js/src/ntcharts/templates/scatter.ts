import { isCategorical, finishCategorical, categorySeries, orientMeasure } from "../category.js";
import { toMs } from "../temporal.js";
import type { ChartTemplateDef } from "flint-chart/core";
import { makeCartesianPivot, makeSortAction } from "flint-chart/core";
import { splitSeries } from "../series.js";
import type { NtInstantiateContext } from "../assemble.js";
import { pinFittedYDomain, pinZeroBaseline } from "../domain.js";

export const ntScatterPlotDef: ChartTemplateDef = {
  chart: "Scatter Plot",
  template: { mark: "circle" },
  channels: ["x", "y", "color", "size", "group"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = orientMeasure(rawCtx as NtInstantiateContext);
    const { emit, channelSemantics: cs, table } = ctx;
    emit.type = "scatter";
    emit.x_axis = { ...emit.x_axis, type: "value", title: cs.x?.field };
    emit.y_axis = { ...emit.y_axis, title: cs.y?.field };
    // Dates are drawn on a time axis: epoch milliseconds, which the scatter's
    // X range fits, labelled as dates.
    const temporal = cs.x?.type === "temporal";
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;
    else if (temporal) emit.x_axis.format = { kind: "time" };
    if (ctx.ntFormatY) emit.y_axis.format = ctx.ntFormatY;
    if (cs.y?.zero?.zero) pinZeroBaseline(emit, cs, table);
    else pinFittedYDomain(emit, cs, table);

    if (isCategorical(ctx)) {
      finishCategorical(ctx, categorySeries(ctx), false);
      return;
    }

    const sizeField = cs.size?.field;
    emit.data.series = splitSeries(table, cs, (row) => {
      const p: any = { x: temporal ? toMs(row[cs.x!.field]) : Number(row[cs.x!.field]), y: Number(row[cs.y!.field]) };
      if (sizeField != null) p.size = Number(row[sizeField]);
      return p;
    });
    if (emit.data.series.length > 1) emit.options = { ...emit.options, show_legend: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({
    transpose: [["x", "y"]],
    permute: [["x", "y", "color", "size"]],
    shift: ["color", "group"],
  }),
};
