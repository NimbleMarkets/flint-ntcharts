import type { ChartTemplateDef } from "flint-chart";
import { toMs } from "../temporal.js";
import type { NtInstantiateContext } from "../assemble.js";
import type { NtOHLCPoint } from "../types.js";

// Palette convention frozen by ntcharts-spec's buildOHLC (candleStyle /
// defaultUpColor / defaultDownColor in ../ntcharts spec/build.go):
// Theme.Palette[0] = up candles (close >= open), Palette[1] = down candles
// (close < open). We always emit both slots explicitly so a consumer never
// silently falls back to the Go-side default.
const OHLC_PALETTE = ["#26a69a", "#ef5350"];

export const ntCandlestickChartDef: ChartTemplateDef = {
  chart: "Candlestick Chart",
  // template.mark only seeds assemble.ts's `markType` (zero-decision +
  // filterOverflow's connected-mark detection), it is not a Vega-Lite
  // skeleton we render. OHLC data is a time-ordered series like a line
  // chart's, so "line" gets the same "connected mark" overflow treatment
  // (contiguous window kept, not arbitrary category sampling) rather than
  // the discrete-mark default.
  template: { mark: "line" },
  channels: ["x", "open", "high", "low", "close"],
  markCognitiveChannel: "position",
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table } = ctx;
    emit.type = "ohlc";

    const xField = cs.x!.field;
    emit.x_axis = { ...emit.x_axis, type: "time", title: xField };
    if (ctx.ntFormatX) emit.x_axis.format = ctx.ntFormatX;

    const openField = cs.open!.field;
    const highField = cs.high!.field;
    const lowField = cs.low!.field;
    const closeField = cs.close!.field;

    const points: NtOHLCPoint[] = table.map((row) => ({
      t: toMs(row[xField]),
      o: Number(row[openField]),
      h: Number(row[highField]),
      l: Number(row[lowField]),
      c: Number(row[closeField]),
    }));

    // Drop points whose x failed to parse (unparsable temporal cell yields
    // NaN from toMs), matching line.ts's invalid-temporal-x idiom, rather
    // than silently plotting a fake epoch-0 candle. o/h/l/c are trusted
    // quantitative fields (not user-facing temporal parsing), so only t is
    // checked.
    let droppedCount = 0;
    const kept = points.filter((p) => {
      const ok = !Number.isNaN(p.t as number);
      if (!ok) droppedCount++;
      return ok;
    });
    // ntcharts' buildOHLC assumes chronological insertion order (candles are
    // drawn column-by-column in Series.OHLC order); sort ascending by t so
    // the drawn candles are correct regardless of input row order.
    kept.sort((a, b) => (a.t as number) - (b.t as number));
    if (droppedCount > 0) {
      ctx.warn({
        severity: "warning",
        code: "invalid-temporal-x",
        message: `dropped ${droppedCount} point(s) with unparsable X values`,
      });
    }

    // Single series: Candlestick Chart has no color/group channel (ntcharts
    // spec.Build's buildOHLC takes the first series with OHLC points, but
    // the frozen emission semantics for this template are single-series).
    emit.data.series = [{ name: xField, ohlc: kept }];
    emit.theme = { ...emit.theme, palette: OHLC_PALETTE };
  },
};
