import { describe, it, expect } from "vitest";
import { assembleNtcharts } from "../src/ntcharts/index.js";
import { binValues } from "../src/ntcharts/bins.js";

const run = (chartType: string, values: any[], encodings: any, extra: any = {}) =>
  assembleNtcharts({ data: { values }, chart_spec: { chartType, encodings, ...extra } } as any);
const codes = (o: any) => (o._warnings ?? []).map((w: any) => w.code);

describe("binValues()", () => {
  it("counts values into equal-width bins, last bin closed", () => {
    const b = binValues([0, 1, 2, 3, 4, 5, 10], 2);
    expect(b.edges).toEqual([0, 5, 10]);
    expect(b.counts).toEqual([5, 2]);
  });
  it("a single distinct value gets one bin holding everything", () => {
    const b = binValues([4, 4, 4], 5);
    expect(b.counts).toEqual([3]);
    expect(b.edges).toHaveLength(2);
  });
  it("no values, no bins", () => {
    expect(binValues([], 5).counts).toEqual([]);
  });
});

describe("Histogram", () => {
  const rows = [1, 2, 2, 3, 3, 3, 4, 4, 5, 9].map((v) => ({ v }));

  it("emits a bar chart of counts per bin with range labels", () => {
    const out = run("Histogram", rows, { x: { field: "v" } }, { chartProperties: { binCount: 4 } });
    expect(out.type).toBe("bar");
    expect(out.x_axis?.labels).toHaveLength(4);
    expect(out.y_axis).toMatchObject({ title: "Count", min: 0 });
    const ys = out.data.series[0].values!.map((p) => p.y);
    expect(ys.reduce((a, b) => a + b, 0)).toBe(10);
    expect(ys).toHaveLength(4);
  });

  it("defaults to 10 bins and never plans more bars than the terminal fits", () => {
    const many = Array.from({ length: 200 }, (_, i) => ({ v: i }));
    expect(run("Histogram", many, { x: { field: "v" } }).x_axis?.labels).toHaveLength(10);
    const narrow = run("Histogram", many, { x: { field: "v" } }, { baseSize: { width: 20, height: 8 }, chartProperties: { binCount: 50 } });
    expect(narrow.x_axis!.labels!.length).toBeLessThanOrEqual(6);
  });

  it("splits by color into stacked series sharing the bins", () => {
    const g = [1, 2, 3, 8, 9].map((v, i) => ({ v, g: i < 3 ? "a" : "b" }));
    const out = run("Histogram", g, { x: { field: "v" }, color: { field: "g" } }, { chartProperties: { binCount: 2 } });
    expect(out.data.series.map((s) => s.name)).toEqual(["a", "b"]);
    expect(out.data.series.map((s) => s.values!.map((p) => p.y))).toEqual([[3, 0], [0, 2]]);
    expect(out.options).toMatchObject({ stacked: true, show_legend: true });
  });

  it("drops non-numbers with a warning", () => {
    const out = run("Histogram", [{ v: 1 }, { v: "x" }, { v: 2 }], { x: { field: "v" } });
    expect(codes(out)).toContain("invalid-value");
    expect(out.data.series[0].values!.reduce((a, p) => a + p.y, 0)).toBe(2);
  });
});

describe("Area Chart", () => {
  const rows = [{ x: 1, y: 2 }, { x: 2, y: 5 }, { x: 3, y: 3 }];
  it("is a line chart, and says the fill is not drawn", () => {
    const out = run("Area Chart", rows, { x: { field: "x" }, y: { field: "y" } });
    expect(out.type).toBe("line");
    const w = (out._warnings ?? []).find((x: any) => x.code === "chart-type-approximated");
    expect(w?.severity).toBe("info");
    expect(w?.message).toMatch(/fill/i);
  });
  it("keeps temporal x as a time series", () => {
    const out = run("Area Chart", [{ x: "2026-01-01", y: 1 }, { x: "2026-02-01", y: 2 }], { x: { field: "x" }, y: { field: "y" } });
    expect(out.type).toBe("timeseries");
  });
});

describe("Lollipop Chart", () => {
  it("is a bar chart, and says the stems are drawn as bars", () => {
    const out = run("Lollipop Chart", [{ c: "a", n: 3 }, { c: "b", n: 5 }], { x: { field: "c" }, y: { field: "n" } });
    expect(out.type).toBe("bar");
    expect(out.x_axis?.labels).toEqual(["a", "b"]);
    const w = (out._warnings ?? []).find((x: any) => x.code === "chart-type-approximated");
    expect(w?.severity).toBe("info");
    expect(w?.message).toMatch(/bar/i);
  });
});
