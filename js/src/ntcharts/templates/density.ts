import type { ChartTemplateDef } from "flint-chart/core";
import { makeCartesianPivot } from "flint-chart/core";
import { rowGroups } from "../groups.js";
import type { NtInstantiateContext } from "../assemble.js";

// Bandwidth as flint (and vega-statistics) estimate it: 1.06 · min(sd, IQR/1.34) · n^-0.2.
function bandwidth(values: number[]): number {
  const n = values.length;
  if (n < 2) return 1;
  const sorted = [...values].sort((a, b) => a - b);
  const mean = values.reduce((a, b) => a + b, 0) / n;
  const sd = Math.sqrt(values.reduce((s, v) => s + (v - mean) ** 2, 0) / n);
  const iqr = sorted[Math.floor((n - 1) * 0.75)] - sorted[Math.floor((n - 1) * 0.25)];
  const v = Math.min(sd, iqr / 1.34 || sd) || sd || 1;
  return 1.06 * v * Math.pow(n, -0.2);
}

/** A Gaussian kernel density estimate at `steps + 1` points across [min, max], as flint computes it. */
export function kde(values: number[], steps: number, multiplier: number, extent: { min: number; max: number }): { x: number; y: number }[] {
  if (!values.length) return [];
  const h = bandwidth(values) * multiplier;
  const span = extent.max - extent.min || 1;
  const norm = values.length * h * Math.sqrt(2 * Math.PI);
  return Array.from({ length: steps + 1 }, (_, i) => {
    const x = extent.min + (i / steps) * span;
    let sum = 0;
    for (const v of values) { const z = (x - v) / h; sum += Math.exp(-0.5 * z * z); }
    return { x, y: sum / norm };
  });
}

// A density plot is a filled curve; ntcharts has no fill, so each group's
// curve is drawn as a line over the shared extent of every group, as flint
// draws grouped densities.
export const ntDensityPlotDef: ChartTemplateDef = {
  chart: "Density Plot",
  template: { mark: "area" },
  channels: ["x", "color", "column", "row"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    const measure = cs.x?.field;
    if (!measure) return;
    const numbers = (rows: Record<string, unknown>[]) => rows.map((r) => Number(r[measure])).filter(Number.isFinite);
    const all = numbers(table);
    if (!all.length) throw new Error(`A Density Plot needs numbers in ${measure}.`);
    const extent = { min: Math.min(...all), max: Math.max(...all) };
    const multiplier = Number(ctx.chartProperties?.bandwidth) > 0 ? Number(ctx.chartProperties!.bandwidth) : 1;
    // Two samples per column: the line chart's braille resolution.
    const steps = Math.min(200, Math.max(50, 2 * emit.width));

    emit.type = "line";
    emit.x_axis = { ...emit.x_axis, type: "value", title: measure };
    emit.y_axis = { ...emit.y_axis, title: "Density", min: 0 };
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;
    emit.data.series = rowGroups(ctx).map((g) => ({
      name: g.name || measure,
      values: kde(numbers(g.rows), steps, multiplier, extent),
      ...(g.color ? { color: g.color } : {}),
    })).filter((s) => s.values.length);
    if (emit.data.series.length > 1) emit.options = { ...emit.options, show_legend: true };
    // Densities are small numbers (0.02 per unit); the default labels would round them all to 0.
    const peak = Math.max(...emit.data.series.flatMap((s) => (s.values ?? []).map((p) => p.y)));
    if (peak > 0) emit.y_axis.format = { kind: "number", precision: Math.max(0, 1 - Math.floor(Math.log10(peak))) };
    ctx.warn({ severity: "info", code: "chart-type-approximated", message: "drawn as lines: the area under each curve is not filled" });
  },
  encodingActions: [],
  pivot: makeCartesianPivot({ permute: [["color", "column"]], shift: ["color"] }),
};
