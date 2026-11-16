import { toMs } from "./temporal.js";

/**
 * The dates of a series that runs at a regular calendar cadence without gaps —
 * year starts a year apart, month starts one or three months apart, or a fixed
 * step (days, weeks, hours) — sorted; undefined otherwise.
 */
export function regularDates(values: unknown[]): number[] | undefined {
  const ms = [...new Set(values.map(toMs).filter(Number.isFinite))].sort((a, b) => a - b);
  if (ms.length < 2) return undefined;
  const d = ms.map((v) => new Date(v));
  const monthIndex = d.map((t) => t.getUTCFullYear() * 12 + t.getUTCMonth());
  const monthStarts = d.every((t) => t.getUTCDate() === 1 && t.getUTCHours() === 0 && t.getUTCMinutes() === 0);
  if (monthStarts) {
    const step = monthIndex[1] - monthIndex[0];
    if (step > 0 && monthIndex.every((m, i) => i === 0 || m - monthIndex[i - 1] === step)) return ms;
  }
  const step = ms[1] - ms[0];
  return ms.every((v, i) => i === 0 || v - ms[i - 1] === step) ? ms : undefined;
}

