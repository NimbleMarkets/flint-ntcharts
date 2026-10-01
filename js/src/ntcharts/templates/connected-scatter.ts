import type { ChartTemplateDef } from "flint-chart/core";
import { makeCartesianPivot } from "flint-chart/core";
import { splitSeries } from "../series.js";
import { toMs } from "../temporal.js";
import type { NtInstantiateContext } from "../assemble.js";
import { pinFittedYDomain } from "../domain.js";

// A connected scatter plot is a path: points are joined in the order channel's
// order (or row order), not sorted by x like a line chart. The spec's line
// builder joins points in input order, so the only job here is ordering.
export const ntConnectedScatterDef: ChartTemplateDef = {
  chart: "Connected Scatter Plot",
  template: { mark: "line" },
  channels: ["x", "y", "order", "color", "detail"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    emit.type = "line";
    emit.x_axis = { ...emit.x_axis, type: "value", title: cs.x?.field };
    emit.y_axis = { ...emit.y_axis, title: cs.y?.field };
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;
    if (ctx.ntFormatY) emit.y_axis.format = ctx.ntFormatY;
    if (cs.y?.zero?.zero) emit.y_axis.min = 0;
    else pinFittedYDomain(emit, cs, table);

    let rows = table;
    const orderField = cs.order?.field;
    if (orderField != null) {
      const key = (r: Record<string, unknown>) =>
        cs.order!.type === "temporal" ? toMs(r[orderField]) : Number(r[orderField]);
      const numeric = table.every((r) => Number.isFinite(key(r)));
      rows = [...table].sort((a, b) =>
        numeric ? key(a) - key(b) : String(a[orderField]).localeCompare(String(b[orderField])),
      );
    }
    const grouping = { ...cs, group: cs.group ?? cs.detail };
    emit.data.series = splitSeries(rows, grouping, (row) => ({
      x: Number(row[cs.x!.field]),
      y: Number(row[cs.y!.field]),
    }));
    if (emit.data.series.length > 1) emit.options = { ...emit.options, show_legend: true };
  },
  encodingActions: [],
  pivot: makeCartesianPivot({ transpose: [["x", "y"]], permute: [["x", "y", "color"]], shift: ["color", "detail"] }),
};
