import { describe, it, expect } from "vitest";
import { assembleNtcharts } from "../src/ntcharts/index.js";
import { calendarGrid } from "../src/ntcharts/calendar.js";

const ms = (s: string) => Date.parse(s + "T00:00:00Z");
const run = (values: any[], encodings: any) =>
  assembleNtcharts({ data: { values }, chart_spec: { chartType: "Calendar Heatmap", encodings } } as any);

describe("calendarGrid()", () => {
  it("puts Mondays in the first row and starts a new column each Monday", () => {
    // 2026-01-05 is a Monday
    const g = calendarGrid([
      { t: ms("2026-01-05"), v: 1 }, // Mon wk0
      { t: ms("2026-01-11"), v: 2 }, // Sun wk0
      { t: ms("2026-01-12"), v: 3 }, // Mon wk1
    ]);
    expect(g.weeks).toBe(2);
    expect(g.cells).toEqual([{ x: 0, y: 0, z: 1 }, { x: 0, y: 6, z: 2 }, { x: 1, y: 0, z: 3 }]);
  });
  it("sums values that fall on the same day", () => {
    const g = calendarGrid([{ t: ms("2026-01-05"), v: 1 }, { t: ms("2026-01-05") + 3600e3, v: 4 }]);
    expect(g.cells).toEqual([{ x: 0, y: 0, z: 5 }]);
  });
  it("keeps empty weeks between data as columns", () => {
    const g = calendarGrid([{ t: ms("2026-01-05"), v: 1 }, { t: ms("2026-01-26"), v: 1 }]);
    expect(g.weeks).toBe(4);
  });
  it("labels the column where a month begins", () => {
    const g = calendarGrid([{ t: ms("2026-01-12"), v: 1 }, { t: ms("2026-02-09"), v: 1 }]);
    // weeks start Jan 12, 19, 26, Feb 2, 9; Feb 1 is a Sunday in the Jan 26 week
    expect(g.labels).toEqual(["Jan", "", "Feb", "", ""]);
  });
});

describe("Calendar Heatmap", () => {
  const days = Array.from({ length: 21 }, (_, i) => ({
    d: new Date(ms("2026-03-02") + i * 86400e3).toISOString().slice(0, 10), n: i,
  }));
  it("emits a heatmap with weekday rows and week columns", () => {
    const out = run(days, { x: { field: "d" }, color: { field: "n" } });
    expect(out.type).toBe("heatmap");
    expect(out.y_axis?.labels).toEqual(["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]);
    expect(out.x_axis?.labels).toHaveLength(3);
    expect(out.heat?.cells).toHaveLength(21);
  });
  it("counts rows per day when there is no color channel", () => {
    const out = run([{ d: "2026-03-02" }, { d: "2026-03-02" }, { d: "2026-03-03" }], { x: { field: "d" } });
    expect(out.heat?.cells).toEqual([{ x: 0, y: 0, z: 2 }, { x: 0, y: 1, z: 1 }]);
  });
  it("skips unparsable dates with a warning", () => {
    const out = run([{ d: "2026-03-02", n: 1 }, { d: "soon", n: 2 }], { x: { field: "d" }, color: { field: "n" } });
    expect(out.heat?.cells).toHaveLength(1);
    expect((out._warnings ?? []).map((w: any) => w.code)).toContain("invalid-temporal-x");
  });
});
