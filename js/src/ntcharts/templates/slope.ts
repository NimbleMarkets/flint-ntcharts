import type { ChartTemplateDef } from "flint-chart/core";
import type { NtInstantiateContext } from "../assemble.js";
import { ntLineChartDef } from "./line.js";

export const ntSlopeChartDef: ChartTemplateDef = {
  ...ntLineChartDef,
  chart: "Slope Chart",
  channels: ["x", "y", "color", "detail"],
  pivot: undefined,
  normalizeEncodings: (enc: any) => ({ ...enc, x: { ...enc.x, type: "ordinal" },
    ...(enc.detail && !enc.color ? { group: enc.detail } : {}) }),
  declareLayoutMode: () => ({ axisFlags: { x: { banded: true } }, paramOverrides: { minStep: 2 } }),
  instantiate: (spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const x = ctx.channelSemantics.x;
    const labels = [...new Set(ctx.table.filter(r => r[x.field] != null).map(r => String(r[x.field])))];
    if (labels.length !== 2) throw new Error("Slope Chart requires exactly two periods");
    if (!x.ordinalSortOrder?.length) {
      if (labels.every(v => v.trim() && Number.isFinite(Number(v)))) labels.sort((a,b) => Number(a)-Number(b));
      else if (labels.every(v => Number.isFinite(Date.parse(v)))) labels.sort((a,b) => Date.parse(a)-Date.parse(b));
      x.ordinalSortOrder = labels;
    }
    if (ctx.chartProperties?.includeZero_y === undefined && ctx.channelSemantics.y.zero) {
      ctx.channelSemantics.y.zero = { ...ctx.channelSemantics.y.zero, zero: false, domainPadFraction: .05 };
    }
    ntLineChartDef.instantiate(spec, rawCtx);
    if (ctx.encodings.detail && !ctx.encodings.color && ctx.emit.options) ctx.emit.options.show_legend = false;
    if (ctx.chartProperties?.showText !== false) ctx.warn({severity: "info", code: "slope-labels-unsupported",
      message: "terminal Slope Chart shows period ticks and series colors; endpoint value labels are not drawn"});
  },
};
