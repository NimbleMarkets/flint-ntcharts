import type { ChartTemplateDef } from "flint-chart/core";
import { ntLineChartDef } from "./line.js";
import type { NtInstantiateContext } from "../assemble.js";

// A streamgraph stacks filled streams around a moving centre. ntcharts has
// neither fill nor stacking, so each stream is drawn as its own line, as the
// terminal draws an area chart.
export const ntStreamgraphDef: ChartTemplateDef = {
  ...ntLineChartDef,
  chart: "Streamgraph",
  template: { mark: "area" },
  channels: ["x", "y", "color"],
  instantiate: (spec: any, rawCtx: any) => {
    ntLineChartDef.instantiate(spec, rawCtx);
    (rawCtx as NtInstantiateContext).warn({
      severity: "info",
      code: "chart-type-approximated",
      message: "drawn as a line chart: each stream is its own line, not stacked or filled",
    });
  },
};
