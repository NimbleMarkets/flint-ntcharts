import { computePaddedDomain } from "flint-chart/core";

// pinFittedYDomain pins emit.y_axis.min/max to a padded data-fitted domain
// when flint's zero-decision EXCLUDES zero. Without the pin, ntcharts'
// terminal linecharts default their view to a 0 baseline (New() starts at
// 0..1 and auto-range only expands), squashing e.g. a 101..121 price series
// into the top rows. Mirrors the upstream ECharts backend's fitted-domain
// branch (instantiate-spec.ts): !zero && domainPadFraction > 0 →
// computePaddedDomain → axis min/max. The zero===true case is handled at
// the call sites (min = 0), matching upstream.
export function pinFittedYDomain(emit: any, cs: any, table: any[]): void {
  const decision = cs.y?.zero;
  const field = cs.y?.field;
  if (!decision || decision.zero || !(decision.domainPadFraction > 0) || field == null) return;
  const values = table
    .map((row: any) => row[field])
    .filter((v: any) => typeof v === "number" && Number.isFinite(v));
  const padded = computePaddedDomain(values, decision.domainPadFraction);
  if (padded) {
    // Padding keeps marks off the frame; it never makes room for a sign
    // the data does not have.
    emit.y_axis.min = values.every((v: number) => v >= 0) ? Math.max(0, padded[0]) : padded[0];
    emit.y_axis.max = values.every((v: number) => v <= 0) ? Math.min(0, padded[1]) : padded[1];
  }
}

// pinZeroBaseline makes the Y axis include zero, as flint's zero decision
// asks: zero becomes the minimum when every value is at or above it, the
// maximum when every value is at or below it, and a range that already
// crosses zero is left to the data. Pinning the minimum at 0 whatever the
// data would push negative values off the plot.
export function pinZeroBaseline(emit: any, cs: any, table: any[]): void {
  const field = cs.y?.field;
  const ys = table.map((r: any) => Number(r[field])).filter(Number.isFinite);
  if (!ys.length || ys.every((y: number) => y >= 0)) emit.y_axis.min = 0;
  else if (ys.every((y: number) => y <= 0)) emit.y_axis.max = 0;
}
