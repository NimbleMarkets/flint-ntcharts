import type { ChartTemplateDef } from "flint-chart/core";
import { detectBandedAxisFromSemantics } from "flint-chart/core";
import { gradientForScheme } from "../colormap.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntHeatmapDef: ChartTemplateDef = {
  chart: "Heatmap",
  template: { mark: "rect" },
  channels: ["x", "y", "color"],
  markCognitiveChannel: "color",
  declareLayoutMode: (cs, table) => {
    const result = detectBandedAxisFromSemantics(cs, table, { preferAxis: "x" });
    return {
      axisFlags: { x: { banded: true }, y: { banded: true } },
      resolvedTypes: result?.resolvedTypes,
    };
  },
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    emit.type = "heatmap";

    const indexOf = (field: string, preferred?: string[]) => {
      const order: string[] = [];
      const seen = new Set<string>();
      if (preferred) for (const v of preferred) { seen.add(String(v)); order.push(String(v)); }
      for (const row of table) {
        const v = String(row[field]);
        if (!seen.has(v)) { seen.add(v); order.push(v); }
      }
      return order;
    };
    const xLabels = indexOf(cs.x!.field, cs.x!.ordinalSortOrder);
    const yLabels = indexOf(cs.y!.field, cs.y!.ordinalSortOrder);
    const xi = new Map(xLabels.map((l, i) => [l, i]));
    const yi = new Map(yLabels.map((l, i) => [l, i]));

    const zField = cs.color!.field;
    emit.heat = {
      cells: table.map((row) => ({
        x: xi.get(String(row[cs.x!.field]))!,
        y: yi.get(String(row[cs.y!.field]))!,
        z: Number(row[zField]),
      })),
    };
    emit.data.series = [];
    emit.x_axis = { ...emit.x_axis, type: "category", labels: xLabels, title: cs.x?.field };
    emit.y_axis = { ...emit.y_axis, labels: yLabels, title: cs.y?.field };
    const scheme = cs.color?.colorScheme;
    emit.theme = { ...emit.theme, gradient: gradientForScheme(scheme?.scheme) };
  },
};
