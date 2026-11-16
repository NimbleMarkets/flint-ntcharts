import type { ChartTemplateDef } from "flint-chart/core";
import { computePaddedDomain, makeCartesianPivot } from "flint-chart/core";
import { rowGroups } from "../groups.js";
import { finishCategorical } from "../category.js";
import { toMs } from "../temporal.js";
import type { NtInstantiateContext } from "../assemble.js";
import type { NtSeries } from "../types.js";

// A range area is a band between a low and a high value. ntcharts has no
// fill, so each band is drawn as its two edges, two lines in one colour, and
// the y axis is fitted to both, as flint fits it.
export const ntRangeAreaChartDef: ChartTemplateDef = {
  chart: "Range Area Chart",
  template: { mark: "area" },
  channels: ["x", "y", "y2", "color", "column", "row"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    const [x, low, high] = [cs.x?.field, cs.y?.field, cs.y2?.field];
    if (!x || !low || !high) throw new Error("A Range Area Chart needs x, y (the low edge) and y2 (the high edge).");
    const kind = cs.x!.type === "temporal" ? "time" : cs.x!.type === "quantitative" ? "value" : "category";
    const xOf = (v: unknown) => (kind === "time" ? toMs(v) : kind === "value" ? Number(v) : v == null ? undefined : String(v));

    const series: NtSeries[] = [];
    for (const g of rowGroups(ctx)) {
      for (const [edge, field] of [["low", low], ["high", high]] as const) {
        const values = g.rows
          .map((r) => ({ x: xOf(r[x]), y: Number(r[field]) }))
          .filter((p) => p.x != null && Number.isFinite(p.y) && !(typeof p.x === "number" && Number.isNaN(p.x)));
        if (kind !== "category") values.sort((a, b) => (a.x as number) - (b.x as number));
        series.push({ name: `${g.name ? `${g.name} ` : ""}${edge === "low" ? low : high}`, values, ...(g.color ? { color: g.color } : { color: "#4e79a7" }) });
      }
    }

    const ys = table.flatMap((r) => [Number(r[low]), Number(r[high])]).filter(Number.isFinite);
    const fitted = computePaddedDomain(ys, cs.y?.zero?.domainPadFraction || 0.05);
    emit.y_axis = { ...emit.y_axis, title: low === high ? low : `${low}, ${high}` };
    if (ctx.ntFormatY) emit.y_axis.format = ctx.ntFormatY;
    if (fitted) [emit.y_axis.min, emit.y_axis.max] = fitted;

    if (kind === "category") {
      emit.type = "line";
      finishCategorical(ctx, series, true);
    } else {
      emit.type = kind === "time" ? "timeseries" : "line";
      emit.x_axis = { ...emit.x_axis, type: kind, title: x };
      if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;
      emit.data.series = series;
    }
    ctx.warn({ severity: "info", code: "chart-type-approximated",
      message: "drawn as two lines per band, its low and high edges: the band between them is not filled" });
  },
  encodingActions: [],
  pivot: makeCartesianPivot({ shift: ["color"] }),
};
