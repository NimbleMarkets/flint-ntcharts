import type { ChartTemplateDef } from "flint-chart/core";
import { makeCartesianPivot } from "flint-chart/core";
import { paletteForScheme } from "../colormap.js";
import { binValues } from "../bins.js";
import type { NtInstantiateContext } from "../assemble.js";

const DEFAULT_BINS = 10;
// Cells a bar chart spends outside its bars (value axis labels and margin).
const AXIS_CELLS = 8;

const num = (v: number) => String(Number(v.toPrecision(3)));

// Histogram bins the measure itself (flint leaves that to the renderer) and
// emits a bar chart of counts, one bar per bin, so it never plans more bins
// than the terminal has room for: the bar model spends two cells per bar.
export const ntHistogramDef: ChartTemplateDef = {
  chart: "Histogram",
  template: { mark: "bar" },
  channels: ["x", "color"],
  markCognitiveChannel: "length",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    const measure = cs.x?.field;
    if (!measure) return;
    const groupField = cs.color?.type === "quantitative" ? undefined : cs.color?.field;

    let dropped = 0;
    const rows = table
      .map((r) => ({ v: Number(r[measure]), g: groupField != null ? String(r[groupField]) : "" }))
      .filter((r) => {
        const ok = Number.isFinite(r.v);
        if (!ok) dropped++;
        return ok;
      });
    if (dropped > 0) {
      ctx.warn({ severity: "warning", code: "invalid-value", message: `dropped ${dropped} value(s) of ${measure} that are not numbers` });
    }

    const requested = Number(ctx.chartProperties?.binCount) > 0 ? Number(ctx.chartProperties!.binCount) : DEFAULT_BINS;
    const capacity = Math.max(1, Math.floor((ctx.canvasSize.width - AXIS_CELLS + 1) / 2));
    const bins = binValues(rows.map((r) => r.v), Math.min(requested, capacity));
    if (requested > capacity && bins.counts.length > 0) {
      ctx.warn({ severity: "info", code: "overflow", message: `using ${bins.counts.length} bins instead of ${requested}: that is all the width fits` });
    }

    const labels = bins.counts.map((_, i) => `${num(bins.edges[i])}–${num(bins.edges[i + 1])}`);
    emit.type = "bar";
    emit.x_axis = { ...emit.x_axis, type: "category", labels, title: measure };
    emit.y_axis = { ...emit.y_axis, title: "Count", min: 0 };

    const groups: string[] = [];
    for (const r of rows) if (!groups.includes(r.g)) groups.push(r.g);
    const palette = paletteForScheme(cs.color?.colorScheme?.scheme);
    emit.data.series = groups.map((g, gi) => {
      const mine = rows.filter((r) => r.g === g).map((r) => r.v);
      // reuse the shared edges so every series lines up with the labels
      const counts = new Array<number>(bins.counts.length).fill(0);
      for (const v of mine) {
        let i = bins.edges.findIndex((e, k) => k < counts.length && v < bins.edges[k + 1]);
        if (i < 0) i = counts.length - 1;
        counts[i]++;
      }
      return {
        name: groupField != null ? g : measure,
        values: counts.map((y) => ({ y })),
        ...(groupField != null ? { color: palette[gi % palette.length] } : {}),
      };
    });
    if (groups.length > 1) emit.options = { ...emit.options, stacked: true, show_legend: true };
  },
  encodingActions: [],
  pivot: makeCartesianPivot({ permute: [["x", "color"]], shift: ["color"] }),
};
