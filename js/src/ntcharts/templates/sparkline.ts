import type { ChartTemplateDef } from "flint-chart/core";
import { toMs } from "../temporal.js";
import type { NtInstantiateContext } from "../assemble.js";
import { ntLineChartDef } from "./line.js";

/**
 * A series spread over `width` columns, one value each: interpolated between
 * neighbours where it has fewer points, each column the mean of its points
 * where it has more. ntcharts' sparkline draws one column per value and keeps
 * only the latest `width` of them, so a series pushed as it is would leave the
 * left of the chart empty, or lose its start.
 */
export function resample(values: number[], width: number): number[] {
  const n = values.length;
  if (n === 0 || width <= 0 || n === width) return values;
  if (n === 1) return new Array(width).fill(values[0]);
  if (n < width) {
    return Array.from({ length: width }, (_, i) => {
      const t = (i * (n - 1)) / (width - 1);
      const lo = Math.floor(t), hi = Math.min(n - 1, lo + 1);
      return values[lo] + (values[hi] - values[lo]) * (t - lo);
    });
  }
  return Array.from({ length: width }, (_, i) => {
    const from = Math.floor((i * n) / width), to = Math.max(from + 1, Math.floor(((i + 1) * n) / width));
    const bucket = values.slice(from, to);
    return bucket.reduce((a, b) => a + b, 0) / bucket.length;
  });
}

export const ntSparklineDef: ChartTemplateDef = {
  chart: "Sparkline",
  // Matches flint-chart's own Sparkline template mark ("line") -- a
  // sparkline is a compressed, connected time/index series, so it gets the
  // same connected-mark overflow treatment as Line Chart (see line.ts).
  template: { mark: "line" },
  channels: ["x", "y", "color"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    // Several series are several sparklines; the terminal draws one, so they
    // are drawn as lines on one small chart rather than run together.
    if (cs.color && cs.color.type !== "quantitative") {
      ntLineChartDef.instantiate(_spec, rawCtx);
      ctx.warn({ severity: "info", code: "chart-type-approximated",
        message: `drawn as a line chart: one sparkline per ${cs.color.field} is not available in the terminal`,
        channel: "color", field: cs.color.field } as any);
      return;
    }
    emit.type = "sparkline";

    const yField = cs.y!.field;
    const xField = cs.x?.field;
    const temporal = cs.x?.type === "temporal";

    // ntcharts-spec's sparkline.Model is a single, y-only pushed series (no
    // per-point X survives the wire contract -- see NtDataPoint.x's
    // "not applicable" comment in types.ts), so row order is all that
    // matters: sort by x (when bound) before dropping it, matching line.ts's
    // temporal-vs-numeric x coercion.
    let rows = table;
    if (xField != null) {
      rows = [...table].sort((a, b) => {
        const ax = temporal ? toMs(a[xField]) : Number(a[xField]);
        const bx = temporal ? toMs(b[xField]) : Number(b[xField]);
        return ax - bx;
      });
    }

    const values = rows.map((row) => Number(row[yField])).filter(Number.isFinite);
    emit.data.series = [{ name: yField, values: resample(values, emit.width).map((y) => ({ y })) }];
  },
};
