import type { ChartTemplateDef } from "flint-chart/core";
import { gradientForScheme } from "../colormap.js";
import { calendarGrid, WEEKDAYS } from "../calendar.js";
import { toMs } from "../temporal.js";
import type { NtInstantiateContext } from "../assemble.js";

// Calendar heatmap: dates down to days, laid out as week columns by weekday
// rows, coloured by the summed value (or by the row count with no color field).
export const ntCalendarHeatmapDef: ChartTemplateDef = {
  chart: "Calendar Heatmap",
  template: { mark: "rect" },
  channels: ["x", "color"],
  markCognitiveChannel: "color",
  declareLayoutMode: () => ({ axisFlags: { x: { banded: true }, y: { banded: true } } }),
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    const dateField = cs.x?.field;
    if (!dateField) return;
    const valueField = cs.color?.field;

    let dropped = 0;
    const points: { t: number; v: number }[] = [];
    for (const row of table) {
      const t = toMs(row[dateField]);
      const v = valueField != null ? Number(row[valueField]) : 1;
      if (Number.isNaN(t) || Number.isNaN(v)) { dropped++; continue; }
      points.push({ t, v });
    }
    if (dropped > 0) {
      ctx.warn({ severity: "warning", code: "invalid-temporal-x", message: `dropped ${dropped} row(s) with an unparsable date or value` });
    }

    const grid = calendarGrid(points);
    emit.type = "heatmap";
    emit.heat = { cells: grid.cells };
    emit.data.series = [];
    emit.x_axis = { ...emit.x_axis, type: "category", labels: grid.labels, title: dateField };
    emit.y_axis = { ...emit.y_axis, labels: WEEKDAYS };
    emit.theme = { ...emit.theme, gradient: gradientForScheme(cs.color?.colorScheme?.scheme) };
  },
};
