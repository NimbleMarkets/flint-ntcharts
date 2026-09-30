import type { ChartTemplateDef } from "flint-chart/core";
import { toMs } from "../temporal.js";
import type { NtInstantiateContext } from "../assemble.js";

export const ntSparklineDef: ChartTemplateDef = {
  chart: "Sparkline",
  // Matches flint-chart's own Sparkline template mark ("line") -- a
  // sparkline is a compressed, connected time/index series, so it gets the
  // same connected-mark overflow treatment as Line Chart (see line.ts).
  template: { mark: "line" },
  channels: ["x", "y"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    emit.type = "sparkline";

    const yField = cs.y!.field;
    const xField = cs.x?.field;
    const temporal = cs.x?.type === "temporal";

    // ntcharts-spec's sparkline.Model is a single, y-only pushed series (no
    // per-point X survives the wire contract -- see NtDataPoint.x's
    // "not applicable" comment in types.ts), so row order is all that
    // matters: sort by x (when bound) before dropping it, matching line.ts's
    // temporal-vs-numeric x coercion.
    let rows = table;
    if (xField != null) {
      rows = [...table].sort((a, b) => {
        const ax = temporal ? toMs(a[xField]) : Number(a[xField]);
        const bx = temporal ? toMs(b[xField]) : Number(b[xField]);
        return ax - bx;
      });
    }

    emit.data.series = [{ name: yField, values: rows.map((row) => ({ y: Number(row[yField]) })) }];
  },
};
