const DAY = 86400e3;
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
export const WEEKDAYS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

const dayStart = (t: number) => Math.floor(t / DAY) * DAY;
// 0 = Monday. 1970-01-01 was a Thursday.
const weekday = (day: number) => (((Math.floor(day / DAY) + 3) % 7) + 7) % 7;

// calendarGrid lays dated values out like a calendar: one column per week
// (starting Monday), one row per weekday, Monday on top. Values on the same
// day are summed, and weeks with no data between the first and last day are
// kept so the time axis stays honest. Columns are labelled with the month
// name in the week the month begins.
export function calendarGrid(points: { t: number; v: number }[]): {
  weeks: number;
  cells: { x: number; y: number; z: number }[];
  labels: string[];
} {
  if (points.length === 0) return { weeks: 0, cells: [], labels: [] };
  const byDay = new Map<number, number>();
  for (const p of points) {
    const d = dayStart(p.t);
    byDay.set(d, (byDay.get(d) ?? 0) + p.v);
  }
  const days = [...byDay.keys()].sort((a, b) => a - b);
  const first = days[0] - weekday(days[0]) * DAY;
  const weeks = Math.floor((days[days.length - 1] - first) / (7 * DAY)) + 1;
  const cells = days.map((d) => ({
    x: Math.floor((d - first) / (7 * DAY)),
    y: weekday(d),
    z: byDay.get(d)!,
  }));
  const labels = Array.from({ length: weeks }, (_, w) => {
    const start = first + w * 7 * DAY;
    // the month whose 1st falls in this week; the first column falls back to
    // the month of the first day with data so the axis never starts blank
    for (let d = 0; d < 7; d++) {
      const date = new Date(start + d * DAY);
      if (date.getUTCDate() === 1) return MONTHS[date.getUTCMonth()];
    }
    return w === 0 ? MONTHS[new Date(days[0]).getUTCMonth()] : "";
  });
  return { weeks, cells, labels };
}
