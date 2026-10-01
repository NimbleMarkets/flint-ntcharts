import type { ChartTemplateDef } from "flint-chart/core";
import { ntLineChartDef } from "./line.js";
import type { NtInstantiateContext } from "../assemble.js";

// ntcharts has no filled area, so an area chart is drawn as its outline: a
// line chart. Stacked areas are not stacked either; each series is its own line.
export const ntAreaChartDef: ChartTemplateDef = {
  ...ntLineChartDef,
  chart: "Area Chart",
  template: { mark: "area" },
  channels: ["x", "y", "color", "opacity"],
  instantiate: (spec: any, rawCtx: any) => {
    ntLineChartDef.instantiate(spec, rawCtx);
    (rawCtx as NtInstantiateContext).warn({
      severity: "info",
      code: "chart-type-approximated",
      message: "drawn as a line chart: the area fill is not shown",
    });
  },
};
