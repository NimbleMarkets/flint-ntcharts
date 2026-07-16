import type { ChartTemplateDef } from "flint-chart";
import { makeCartesianPivot, makeSortAction } from "flint-chart";
import { splitSeries } from "../series.js";
import type { NtInstantiateContext } from "../assemble.js";

// toMs coerces a converted temporal cell to ms-since-epoch, matching
// ntcharts-spec's time convention. Verified shape (see task-4 report): rows
// from convertTemporalData/filterOverflow carry temporal cells as canonical
// ISO date strings (e.g. "2026-01-01"), which Date.parse handles; the
// Date/number branches are defensive for other call sites/future inputs.
function toMs(v: unknown): number {
  if (v instanceof Date) return v.getTime();
  if (typeof v === "number") return v;
  const t = Date.parse(String(v));
  return Number.isNaN(t) ? 0 : t;
}

export const ntLineChartDef: ChartTemplateDef = {
  chart: "Line Chart",
  template: { mark: "line" },
  channels: ["x", "y", "color", "group"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    const temporal = cs.x?.type === "temporal";
    emit.type = temporal ? "timeseries" : "line";
    emit.x_axis = { ...emit.x_axis, type: temporal ? "time" : "value", title: cs.x?.field };
    emit.y_axis = { ...emit.y_axis, title: cs.y?.field };
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;
    if (ctx.ntFormatY) emit.y_axis.format = ctx.ntFormatY;
    if (cs.y?.zero?.zero) emit.y_axis.min = 0;

    const xField = cs.x!.field;
    emit.data.series = splitSeries(table, cs, (row) => ({
      x: temporal ? toMs(row[xField]) : Number(row[xField]),
      y: Number(row[cs.y!.field]),
    }));
    if (emit.data.series.length > 1) emit.options = { ...emit.options, show_legend: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({ permute: [["x", "y", "color"]], shift: ["color", "group"] }),
};
