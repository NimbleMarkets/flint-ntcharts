import type { ChartTemplateDef } from "flint-chart/core";
import { makeCartesianPivot } from "flint-chart/core";
import { categoryKey, categoryOrder } from "../bands.js";
import type { NtInstantiateContext } from "../assemble.js";

type Row = Record<string, unknown>;

// A pyramid sets two groups back to back along one category axis: the first
// group's bars run left from the axis, the second's right, on an axis
// symmetric about zero, as flint draws it. The terminal's horizontal bars do
// that with the first group negated; the first category sits at the bottom,
// as on flint's category y axis. The terminal draws no legend, so the chart
// names its sides in a warning.
export const ntPyramidChartDef: ChartTemplateDef = {
  chart: "Pyramid Chart",
  template: { mark: "bar" },
  channels: ["x", "y", "color"],
  markCognitiveChannel: "length",
  declareLayoutMode: () => ({ axisFlags: { y: { banded: true } }, paramOverrides: { minStep: 2 } }),
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table, encodings } = ctx;
    const yDiscrete = cs.y?.type === "nominal" || cs.y?.type === "ordinal";
    const catCS = yDiscrete ? cs.y : cs.x;
    const valCS = yDiscrete ? cs.x : cs.y;
    if (!catCS || !valCS || valCS.type !== "quantitative") {
      throw new Error("A Pyramid Chart needs a category axis and a quantitative axis.");
    }
    const groupCS = cs.color ?? cs.group;
    const sum = (rows: Row[]) => rows.reduce((s, r) => s + (Number(r[valCS.field]) || 0), 0);
    const keys = categoryOrder(table, catCS, encodings[yDiscrete ? "y" : "x"], sum).reverse();

    const groups: string[] = [];
    if (groupCS) for (const r of table) { const g = String(r[groupCS.field]); if (!groups.includes(g)) groups.push(g); }
    const [left, right] = groups.length ? [groups[0], groups[1] ?? groups[0]] : [valCS.field, valCS.field];
    const totals = (group: string) => {
      const by = new Map<string, number>();
      for (const r of table) {
        if (groupCS && String(r[groupCS.field]) !== group) continue;
        const k = categoryKey(r[catCS.field]);
        by.set(k, (by.get(k) ?? 0) + (Number(r[valCS.field]) || 0));
      }
      return keys.map((k) => by.get(k) ?? 0);
    };
    const leftValues = totals(left).map((v) => -v);
    const rightValues = totals(right);
    const extent = Math.max(0, ...leftValues.map(Math.abs), ...rightValues.map(Math.abs));

    emit.type = "bar";
    emit.options = { ...emit.options, orientation: "horizontal", stacked: true };
    emit.x_axis = { ...emit.x_axis, type: "category", labels: keys, title: catCS.field };
    emit.y_axis = { ...emit.y_axis, title: valCS.field, ...(extent > 0 ? { min: -extent, max: extent } : {}) };
    emit.data.series = [
      { name: left, values: leftValues.map((y) => ({ y })), color: "#4e79a7" },
      { name: right, values: rightValues.map((y) => ({ y })), color: "#f28e2c" },
    ];
    ctx.warn({ severity: "info", code: "pyramid-sides",
      message: groupCS ? `${left} runs left, ${right} right` : `${valCS.field} is drawn on both sides`,
      channel: "color", field: groupCS?.field } as any);
    if (groups.length > 2) {
      ctx.warn({ severity: "warning", code: "too-many-groups-pyramid",
        message: `a pyramid sets two groups back to back; showing ${left} and ${right} of ${groups.length}`,
        channel: "color", field: groupCS!.field } as any);
    }
    if (table.some((r) => Number(r[valCS.field]) < 0)) {
      ctx.warn({ severity: "warning", code: "negative-values-pyramid",
        message: `${valCS.field} has negative values, which cross to the other side`, field: valCS.field } as any);
    }
  },
  encodingActions: [],
  pivot: makeCartesianPivot({ transpose: [["x", "y"]] }),
};
