import type { ChartTemplateDef } from "flint-chart";
import { makeCartesianPivot, makeSortAction } from "flint-chart";
import { ntBarChartDef } from "./bar.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntStackedBarChartDef: ChartTemplateDef = {
  ...ntBarChartDef,
  chart: "Stacked Bar Chart",
  instantiate: (spec: any, rawCtx: any) => {
    ntBarChartDef.instantiate(spec, rawCtx);
    const ctx = rawCtx as NtInstantiateContext;
    ctx.emit.options = { ...ctx.emit.options, stacked: true };
  },
  encodingActions: [makeSortAction()],
  pivot: makeCartesianPivot({
    transpose: [["x", "y"]],
    permute: [["x", "y", "color"]],
    shift: ["color", "group"],
  }),
};
