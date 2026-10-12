import type { ChartTemplateDef } from "flint-chart/core";
import type { NtInstantiateContext } from "../assemble.js";
import { ntScatterPlotDef } from "./scatter.js";

export const ntStripPlotDef: ChartTemplateDef = {
  ...ntScatterPlotDef,
  chart: "Strip Plot",
  pivot: undefined,
  normalizeEncodings: (enc: any) => {
    const discrete = (e: any) => ["nominal", "ordinal"].includes(e?.type);
    if (!discrete(enc.x) && discrete(enc.y)) return { ...enc, x: enc.y, y: enc.x };
    return enc;
  },
  declareLayoutMode: () => ({ axisFlags: { x: { banded: true } }, paramOverrides: { minStep: 2 } }),
  instantiate: (spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    if (!["nominal", "ordinal"].includes(ctx.channelSemantics.x?.type) || ctx.channelSemantics.y?.type !== "quantitative") {
      throw new Error("Strip Plot requires one categorical axis and one quantitative axis");
    }
    ntScatterPlotDef.instantiate(spec, rawCtx);
    if (ctx.chartProperties?.jitterWidth !== 0) ctx.warn({severity: "info",code: "strip-jitter-unsupported",
      message: "terminal Strip Plot aligns observations at category centers; jitter and point-size encoding are not drawn"});
  },
};
