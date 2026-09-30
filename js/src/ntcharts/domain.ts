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
    emit.y_axis.min = padded[0];
    emit.y_axis.max = padded[1];
  }
}
