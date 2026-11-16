import type { ChartTemplateDef } from "flint-chart/core";
import { ntScatterPlotDef } from "./scatter.js";
import type { NtInstantiateContext } from "../assemble.js";

// A ranged dot plot marks two values per category and joins them. The
// terminal draws the dots (the categories along x, as its scatter measures
// along y) but cannot join them.
export const ntRangedDotPlotDef: ChartTemplateDef = {
  ...ntScatterPlotDef,
  chart: "Ranged Dot Plot",
  channels: ["x", "y", "color"],
  instantiate: (spec: any, rawCtx: any) => {
    ntScatterPlotDef.instantiate(spec, rawCtx);
    (rawCtx as NtInstantiateContext).warn({
      severity: "info",
      code: "chart-type-approximated",
      message: "drawn as dots: the line joining each category's dots is not shown",
    });
  },
};
