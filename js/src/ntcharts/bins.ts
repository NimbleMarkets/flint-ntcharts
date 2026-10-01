// binValues counts values into `n` equal-width bins over [min, max]. Every bin
// is half-open except the last, which includes the maximum. A single distinct
// value gets one unit-wide bin.
export function binValues(values: number[], n: number): { edges: number[]; counts: number[] } {
  if (values.length === 0) return { edges: [], counts: [] };
  const min = Math.min(...values);
  const max = Math.max(...values);
  if (min === max) return { edges: [min, min + 1], counts: [values.length] };
  const bins = Math.max(1, Math.floor(n));
  const width = (max - min) / bins;
  const edges = Array.from({ length: bins + 1 }, (_, i) => (i === bins ? max : min + i * width));
  const counts = new Array<number>(bins).fill(0);
  for (const v of values) counts[Math.min(bins - 1, Math.floor((v - min) / width))]++;
  return { edges, counts };
}
