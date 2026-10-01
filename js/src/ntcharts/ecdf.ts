// ecdf turns raw values into the points of an empirical CDF: each distinct
// value paired with the share of values at or below it. Ties collapse to the
// share reached after the last of them, so the curve has one point per step.
export function ecdf(values: number[]): { x: number; y: number }[] {
  const sorted = [...values].sort((a, b) => a - b);
  const n = sorted.length;
  const out: { x: number; y: number }[] = [];
  for (let i = 0; i < n; i++) {
    if (i + 1 < n && sorted[i + 1] === sorted[i]) continue;
    out.push({ x: sorted[i], y: (i + 1) / n });
  }
  return out;
}
