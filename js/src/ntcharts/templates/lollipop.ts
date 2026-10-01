import type { ChartTemplateDef } from "flint-chart/core";
import { ntBarChartDef } from "./bar.js";
import type { NtInstantiateContext } from "../assemble.js";

// A lollipop is a bar with a dot for a head; the terminal draws the bar.
export const ntLollipopChartDef: ChartTemplateDef = {
  ...ntBarChartDef,
  chart: "Lollipop Chart",
  channels: ["x", "y", "color"],
  instantiate: (spec: any, rawCtx: any) => {
    ntBarChartDef.instantiate(spec, rawCtx);
    (rawCtx as NtInstantiateContext).warn({
      severity: "info",
      code: "chart-type-approximated",
      message: "drawn as a bar chart: the lollipop stems and dots are not shown",
    });
  },
};
