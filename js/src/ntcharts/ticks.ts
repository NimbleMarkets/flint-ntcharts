// Which categories an axis names, and with what text, in the fewest
// characters that still say where you are: the terminal's answer to Vega's
// tick count and label-overlap rules.
//
// - Numbers are named at "nice" values (multiples of 1, 2, 5 × 10^k of their
//   step), as many as fit.
// - Dates are named where a calendar period starts, the finest period whose
//   names fit: hours (`06`), days (`8`), weeks, months (`Apr`), quarters,
//   half-years, years (`2021`, `'21`), decades. Where the next larger period
//   also starts, that period's name stands in (`Mar` for a month on a day
//   axis, `2021` for a year on a month axis).
// - Ordered categories are named every k-th, the smallest k that fits.
// - Nominal categories are all named: one that does not fit its room is cut
//   with an ellipsis (`Electr…`). Every name matters, so none is skipped.
//
// The geometry mirrors ntcharts': a bar's name starts at its bar and may run
// on under unnamed bars; a category slot's name is centred on the slot and
// may use the columns halfway to the next named slot.

export type TickKind = "date" | "number" | "ordinal" | "nominal";

/** Where each category's name can start, and how wide the axis is, in columns. */
export interface Slots {
  cols: number[];
  width: number;
  anchor: "start" | "center";
}

export interface TickPlan {
  /** The text drawn under each category; "" draws nothing. */
  ticks: string[];
  /** Some names were cut with an ellipsis or left out (nominal axes only). */
  shortened: boolean;
}

/** Vertical bars as ntcharts lays them out: equal widths and one-column gaps. */
export function barSlots(n: number, width: number): Slots {
  const bar = Math.max(1, Math.floor((width - (n - 1)) / Math.max(1, n)));
  return { cols: Array.from({ length: n }, (_, i) => i * (bar + 1)), width, anchor: "start" };
}

/** Category slots of a line or scatter chart (ntcharts' categoryCenters). */
export function centerSlots(n: number, width: number): Slots {
  const cols = Array.from({ length: n }, (_, i) => {
    const left = Math.floor((i * width) / n), right = Math.floor(((i + 1) * width) / n);
    return Math.min(width - 1, Math.floor((left + Math.max(left, right - 1)) / 2));
  });
  return { cols, width, anchor: "center" };
}

const textWidth = (s: string) => [...s].length;

/** The room each named category has, given which are named. */
function rooms(ticks: string[], slots: Slots): number[] {
  const named = ticks.map((t, i) => (t ? i : -1)).filter((i) => i >= 0);
  const room = new Array<number>(ticks.length).fill(0);
  named.forEach((i, k) => {
    const c = slots.cols[i];
    if (slots.anchor === "start") {
      const next = k + 1 < named.length ? slots.cols[named[k + 1]] - 1 : slots.width;
      room[i] = next - c;
    } else {
      const lo = k > 0 ? Math.floor((slots.cols[named[k - 1]] + c) / 2) + 1 : 0;
      const hi = k + 1 < named.length ? Math.floor((c + slots.cols[named[k + 1]] + 1) / 2) - 1 : slots.width - 1;
      room[i] = hi - lo + 1;
    }
  });
  return room;
}

const fits = (ticks: string[], slots: Slots) => {
  const room = rooms(ticks, slots);
  return ticks.every((t, i) => !t || textWidth(t) <= room[i]);
};

/** Cut a name to `room` columns with an ellipsis; too narrow for one, nothing. */
export function ellipsize(s: string, room: number): string {
  const chars = [...s];
  if (chars.length <= room) return s;
  return room >= 2 ? chars.slice(0, room - 1).join("") + "…" : "";
}

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
const DAY = 86_400_000;
const pad2 = (n: number) => String(n).padStart(2, "0");

interface Period {
  /** The shortest this period runs, to skip periods finer than the data. */
  span: number;
  /** Which period a date falls in. */
  bucket: (d: Date) => string;
  /** Its name, and its name where the next larger period starts. */
  minor: (d: Date) => string;
  major: (d: Date) => string;
  /** Which next-larger period a date falls in; a date that starts one is named by it. */
  majorBucket: (d: Date) => string;
  /** Whether a date is exactly where a next-larger period starts (for the first date). */
  startsMajor: (d: Date) => boolean;
}

const Y = (d: Date) => d.getUTCFullYear(), M = (d: Date) => d.getUTCMonth(), D = (d: Date) => d.getUTCDate(), H = (d: Date) => d.getUTCHours();
const yearName = (d: Date) => String(Y(d));
const hours = (k: number): Period => ({
  span: k * 3_600_000, bucket: (d) => `${Y(d)}-${M(d)}-${D(d)}-${Math.floor(H(d) / k)}`,
  minor: (d) => pad2(H(d)), major: (d) => `${M(d) + 1}/${D(d)}`,
  majorBucket: (d) => `${Y(d)}-${M(d)}-${D(d)}`, startsMajor: (d) => H(d) === 0,
});
const daysOfMonth = (span: number, starts: number[]): Period => ({
  span, bucket: (d) => `${Y(d)}-${M(d)}-${starts.filter((s) => D(d) >= s).length}`,
  minor: (d) => String(D(d)), major: (d) => MONTHS[M(d)],
  majorBucket: (d) => `${Y(d)}-${M(d)}`, startsMajor: (d) => D(d) === 1,
});
const months = (k: number, letter = false): Period => ({
  span: k * 28 * DAY, bucket: (d) => `${Y(d)}-${Math.floor(M(d) / k)}`,
  minor: (d) => (letter ? MONTHS[M(d)][0] : MONTHS[M(d)]), major: (d) => (letter ? `'${String(Y(d)).slice(2)}` : yearName(d)),
  majorBucket: (d) => String(Y(d)), startsMajor: (d) => M(d) === 0 && D(d) === 1,
});
const years = (k: number, short = false): Period => ({
  span: k * 365 * DAY, bucket: (d) => String(Math.floor(Y(d) / k)),
  minor: (d) => (short ? `'${String(Y(d)).slice(2)}` : yearName(d)), major: (d) => (short ? `'${String(Y(d)).slice(2)}` : yearName(d)),
  majorBucket: () => "", startsMajor: () => false,
});

/** Calendar periods, finest first; each axis takes the finest that fits. */
const PERIODS: Period[] = [
  hours(1), hours(3), hours(6), hours(12),
  daysOfMonth(DAY, Array.from({ length: 31 }, (_, i) => i + 1)),
  daysOfMonth(7 * DAY, [1, 8, 15, 22]), daysOfMonth(14 * DAY, [1, 15]),
  months(1), months(3), months(6), months(1, true),
  years(1), years(1, true), years(2), years(5), years(10), years(20), years(50), years(100),
];

function dateTicks(dates: Date[], slots: Slots): string[] | undefined {
  const times = dates.map((d) => d.getTime());
  let gap = Infinity;
  for (let i = 1; i < times.length; i++) if (times[i] > times[i - 1]) gap = Math.min(gap, times[i] - times[i - 1]);
  for (const p of PERIODS) {
    if (p.span < gap * 0.9) continue;
    const ticks = dates.map((d, i) => {
      if (i > 0 && p.bucket(d) === p.bucket(dates[i - 1])) return "";
      const major = i === 0 ? p.startsMajor(d) : p.majorBucket(d) !== p.majorBucket(dates[i - 1]);
      return major ? p.major(d) : p.minor(d);
    });
    if (fits(ticks, slots)) return ticks;
  }
  return undefined;
}

const NICE = [1, 2, 2.5, 5];

function numberTicks(values: number[], display: string[], slots: Slots): string[] | undefined {
  const step = values.length > 1 ? values[1] - values[0] : 0;
  const even = step > 0 && values.every((v, i) => i === 0 || Math.abs(v - values[i - 1] - step) <= Math.abs(step) * 1e-6);
  if (!even) return undefined;
  for (let exp = Math.floor(Math.log10(step)); exp <= Math.ceil(Math.log10(Math.abs(values[values.length - 1]) + step)) + 1; exp++) {
    for (const m of NICE) {
      const stride = m * 10 ** exp;
      const ratio = stride / step;
      if (stride < step || Math.abs(ratio - Math.round(ratio)) > 1e-6) continue;
      const on = (v: number) => Math.abs(v / stride - Math.round(v / stride)) < 1e-6;
      if (!values.some(on)) continue;
      const ticks = values.map((v, i) => (on(v) ? display[i] : ""));
      if (fits(ticks, slots)) return ticks;
    }
  }
  return undefined;
}

function everyKth(display: string[], slots: Slots, from = 1): string[] | undefined {
  for (let k = from; k <= display.length; k++) {
    const ticks = display.map((t, i) => (i % k === 0 ? t : ""));
    if (fits(ticks, slots)) return ticks;
  }
  return undefined;
}

const DATE = /^\d{4}-\d{2}(-\d{2})?([T ]\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:?\d{2})?)?$/;

/** A date key in UTC: a date-time without a zone is read as UTC, as Flint reads it. */
export function utcDate(key: string): Date {
  let k = key.length === 7 ? `${key}-01` : key;
  if (k.length > 10 && !/(Z|[+-]\d{2}:?\d{2})$/.test(k)) k = `${k.replace(" ", "T")}Z`;
  return new Date(Date.parse(k));
}

const CLOCK = /^(\d{1,2}):(\d{2})(:\d{2})?$/;

/** Hours of the day named every 1, 2, 3, 4, 6 or 12 hours (`00 06 12 18`). */
function clockTicks(keys: string[], slots: Slots): string[] | undefined {
  const parts = keys.map((k) => CLOCK.exec(k)!);
  const minutes = parts.map((m) => Number(m[1]) * 60 + Number(m[2]));
  const onHour = parts.every((m) => m[2] === "00");
  const text = (i: number) => (onHour ? parts[i][1].padStart(2, "0") : `${parts[i][1].padStart(2, "0")}:${parts[i][2]}`);
  for (const h of [1, 2, 3, 4, 6, 12]) {
    const ticks = minutes.map((m, i) => (m % (h * 60) === 0 ? text(i) : ""));
    if (ticks.some(Boolean) && fits(ticks, slots)) return ticks;
  }
  return undefined;
}

/** What kind of axis a list of category keys makes. */
export function tickKind(keys: string[], ordered: boolean): TickKind {
  const ascending = (xs: number[]) => xs.every((x, i) => i === 0 || x > xs[i - 1]);
  if (keys.length > 1 && keys.every((k) => DATE.test(k)) && ascending(keys.map((k) => utcDate(k).getTime()))) return "date";
  if (keys.length > 1 && keys.every((k) => k.trim() !== "" && Number.isFinite(Number(k))) && ascending(keys.map(Number))) return "number";
  if (keys.length > 1 && keys.every((k) => CLOCK.test(k)) && ascending(keys.map((k) => Number(k.replace(":", "."))))) return "ordinal";
  return ordered ? "ordinal" : "nominal";
}

/**
 * The names to draw under `keys` (displayed as `display`), laid out in
 * `slots`. `stride`, when given, names every stride-th category of an
 * ordered axis instead of choosing.
 */
export function planTicks(keys: string[], display: string[], kind: TickKind, slots: Slots, stride?: number): TickPlan {
  // Dates are named by the calendar (`2021 Apr Jul`), however much room.
  if (kind === "date" && !stride) {
    const named = dateTicks(keys.map(utcDate), slots);
    if (named) return { ticks: named, shortened: false };
  }
  if (fits(display, slots) && !stride) return { ticks: display, shortened: false };
  if (kind !== "nominal") {
    if (stride && stride >= 1) {
      const ticks = display.map((t, i) => (i % Math.round(stride) === 0 ? t : ""));
      return { ticks: ellipsizeAll(ticks, slots), shortened: false };
    }
    const planned =
      (kind === "number" && numberTicks(keys.map(Number), display, slots)) ||
      (keys.every((k) => CLOCK.test(k)) && clockTicks(keys, slots)) ||
      everyKth(display, slots);
    if (planned) return { ticks: planned, shortened: false };
  }
  return { ticks: ellipsizeAll(display, slots), shortened: true };
}

function ellipsizeAll(ticks: string[], slots: Slots): string[] {
  const room = rooms(ticks, slots);
  return ticks.map((t, i) => (t ? ellipsize(t, room[i]) : ""));
}
