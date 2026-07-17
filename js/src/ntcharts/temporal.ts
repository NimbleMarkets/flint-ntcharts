// toMs coerces a converted temporal cell to ms-since-epoch, matching
// ntcharts-spec's time convention. Verified shape (see task-4 report): rows
// from convertTemporalData/filterOverflow carry temporal cells as canonical
// ISO date strings (e.g. "2026-01-01"), which Date.parse handles; the
// Date/number branches are defensive for other call sites/future inputs.
// Unparsable input yields NaN (never a fake epoch-0 point) so callers can
// detect and drop it explicitly.
//
// Shared by line.ts (timeseries x) and candlestick.ts (ohlc t) — both need
// the identical ms-coercion + NaN-on-failure contract, so this lives in one
// place rather than being duplicated or imported cross-template.
export function toMs(v: unknown): number {
  if (v instanceof Date) return v.getTime();
  if (typeof v === "number") return v;
  return Date.parse(String(v));
}
