import type { ChartTemplateDef } from "flint-chart/core";
import { makeCartesianPivot } from "flint-chart/core";
import { splitSeries } from "../series.js";
import { ecdf } from "../ecdf.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntEcdfPlotDef: ChartTemplateDef = {
  chart: "ECDF Plot",
  template: { mark: "line" },
  channels: ["x", "color", "detail"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    const measure = cs.x?.field;
    if (!measure) return;
    // splitSeries groups by color, else group; an ECDF's second grouping
    // channel is called detail.
    const grouping = { ...cs, group: cs.group ?? cs.detail };
    emit.type = "line";
    emit.x_axis = { ...emit.x_axis, type: "value", title: measure };
    emit.y_axis = { ...emit.y_axis, title: "Cumulative proportion", min: 0, max: 1, format: { kind: "percent", precision: 0 } };
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;

    let dropped = 0;
    const raw = splitSeries(table, grouping, (row) => ({ x: Number(row[measure]), y: 0 }));
    emit.data.series = raw.map((s) => {
      const xs = (s.values ?? []).map((p) => p.x as number);
      const kept = xs.filter((x) => Number.isFinite(x));
      dropped += xs.length - kept.length;
      return { ...s, name: s.name === "value" ? measure : s.name, values: ecdf(kept) };
    });
    if (dropped > 0) {
      ctx.warn({
        severity: "warning",
        code: "invalid-value",
        message: `dropped ${dropped} value(s) of ${measure} that are not numbers`,
      });
    }
    if (emit.data.series.length > 1) emit.options = { ...emit.options, show_legend: true };
  },
  encodingActions: [],
  pivot: makeCartesianPivot({ permute: [["color", "detail"]], shift: ["color", "detail"] }),
};
