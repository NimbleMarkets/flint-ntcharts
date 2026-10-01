import type { ChartTemplateDef } from "flint-chart/core";
import { ntScatterPlotDef } from "./scatter.js";
import type { NtInstantiateContext } from "../assemble.js";

// ntcharts scatter plots have no per-point size, so a bubble chart renders as
// a plain scatter plot and says so when a size channel was asked for.
export const ntBubbleChartDef: ChartTemplateDef = {
  ...ntScatterPlotDef,
  chart: "Bubble Chart",
  channels: ["x", "y", "size", "color", "opacity"],
  instantiate: (spec: any, rawCtx: any) => {
    ntScatterPlotDef.instantiate(spec, rawCtx);
    const ctx = rawCtx as NtInstantiateContext;
    const size = ctx.channelSemantics.size?.field;
    if (size != null) {
      ctx.warn({
        severity: "info",
        code: "chart-type-approximated",
        message: `drawn as a scatter plot: the size channel (${size}) is not shown`,
      });
    }
  },
};
