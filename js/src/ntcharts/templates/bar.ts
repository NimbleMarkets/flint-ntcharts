import type { ChartTemplateDef } from "flint-chart";
import { detectBandedAxisFromSemantics, makeCartesianPivot, makeSortAction } from "flint-chart";
import { splitSeries } from "../series.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntBarChartDef: ChartTemplateDef = {
  chart: "Bar Chart",
  template: { mark: "bar" },
  channels: ["x", "y", "color", "group"],
  markCognitiveChannel: "length",
  declareLayoutMode: (cs, table) => {
    const result = detectBandedAxisFromSemantics(cs, table, { preferAxis: "x" });
    return {
      axisFlags: result ? { [result.axis]: { banded: true } } : { x: { banded: true } },
      resolvedTypes: result?.resolvedTypes,
    };
  },
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    // Orientation: banded/nominal axis on y ⇒ horizontal bars.
    const horizontal = cs.y != null && cs.y.type !== "quantitative" && cs.x?.type === "quantitative";
    const catCS = horizontal ? cs.y! : cs.x!;
    const valCS = horizontal ? cs.x! : cs.y!;

    emit.type = "bar";
    if (horizontal) emit.options = { ...emit.options, orientation: "horizontal" };

    // Category labels in table order (post-overflow), deduped.
    const labels: string[] = [];
    const seen = new Set<string>();
    for (const row of table) {
      const v = String(row[catCS.field]);
      if (!seen.has(v)) { seen.add(v); labels.push(v); }
    }
    emit.x_axis = { ...emit.x_axis, type: "category", labels, title: catCS.field };
    emit.y_axis = { ...emit.y_axis, title: valCS.field };
    const valueFormat = horizontal ? ctx.ntFormatX : ctx.ntFormatY;
    if (valueFormat) emit.y_axis.format = valueFormat;
    if (valCS.zero?.zero) emit.y_axis.min = 0;

    // Values aligned to the label order; series split by color.
    const index = new Map(labels.map((l, i) => [l, i]));
    const series = splitSeries(table, cs, (row) => ({ y: Number(row[valCS.field]) }));
    const groupField = cs.color?.field ?? cs.group?.field;
    for (const s of series) {
      const slots = labels.map(() => ({ y: 0 }));
      for (const row of table) {
        const cat = String(row[catCS.field]);
        const belongs = series.length === 1 || groupField == null || String(row[groupField]) === s.name;
        if (belongs) slots[index.get(cat)!] = { y: Number(row[valCS.field]) };
      }
      s.values = slots;
    }
    emit.data.series = series;
    if (series.length > 1) emit.options = { ...emit.options, stacked: true, show_legend: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({
    transpose: [["x", "y"]],
    permute: [["x", "y", "color"]],
    shift: ["color", "group"],
  }),
};
