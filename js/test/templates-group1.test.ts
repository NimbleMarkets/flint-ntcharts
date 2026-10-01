import { describe, it, expect } from "vitest";
import { assembleNtcharts } from "../src/ntcharts/index.js";
import { ecdf } from "../src/ntcharts/ecdf.js";

const run = (chartType: string, values: any[], encodings: any, extra: any = {}) =>
  assembleNtcharts({ data: { values }, chart_spec: { chartType, encodings, ...extra } } as any);
const codes = (o: any) => (o._warnings ?? []).map((w: any) => w.code);

describe("ecdf()", () => {
  it("sorts and gives each distinct value its cumulative share", () => {
    expect(ecdf([3, 1, 2])).toEqual([{ x: 1, y: 1 / 3 }, { x: 2, y: 2 / 3 }, { x: 3, y: 1 }]);
  });
  it("collapses ties to the share reached after the last of them", () => {
    expect(ecdf([1, 2, 2, 3])).toEqual([{ x: 1, y: 0.25 }, { x: 2, y: 0.75 }, { x: 3, y: 1 }]);
  });
  it("one value is a single point at 100%", () => {
    expect(ecdf([7])).toEqual([{ x: 7, y: 1 }]);
  });
  it("no values, no points", () => {
    expect(ecdf([])).toEqual([]);
  });
});

describe("ECDF Plot", () => {
  const rows = [3, 1, 2, 2, 9, 4].map((v) => ({ v }));

  it("emits a line of cumulative share over 0..100%", () => {
    const out = run("ECDF Plot", rows, { x: { field: "v" } });
    expect(out.type).toBe("line");
    expect(out.x_axis).toMatchObject({ type: "value", title: "v" });
    expect(out.y_axis).toMatchObject({ min: 0, max: 1, format: { kind: "percent", precision: 0 } });
    const pts = out.data.series[0].values!;
    expect(pts.map((p) => p.x)).toEqual([1, 2, 3, 4, 9]);
    expect(pts.map((p) => p.y)).toEqual([1 / 6, 3 / 6, 4 / 6, 5 / 6, 1]);
    expect(codes(out)).not.toContain("chart-type-approximated");
  });

  it("one cumulative line per color", () => {
    const grouped = [
      ...[1, 2, 3].map((v) => ({ v, g: "a" })),
      ...[10, 20].map((v) => ({ v, g: "b" })),
    ];
    const out = run("ECDF Plot", grouped, { x: { field: "v" }, color: { field: "g" } });
    expect(out.data.series.map((s) => s.name)).toEqual(["a", "b"]);
    expect(out.data.series[0].values!.at(-1)!.y).toBe(1);
    expect(out.data.series[1].values!.map((p) => p.y)).toEqual([0.5, 1]);
    expect(out.options?.show_legend).toBe(true);
  });

  it("drops values that are not numbers and says so", () => {
    const out = run("ECDF Plot", [{ v: 1 }, { v: "n/a" }, { v: 3 }], { x: { field: "v" } });
    expect(out.data.series[0].values!.map((p) => p.x)).toEqual([1, 3]);
    expect(codes(out)).toContain("invalid-value");
  });

  it("honours a log X axis", () => {
    const out = run("ECDF Plot", [1, 10, 100, 1000].map((v) => ({ v })), { x: { field: "v" } }, { chartProperties: { logScale_x: true } });
    expect(out.x_axis?.scale).toBe("log");
  });
});

describe("Connected Scatter Plot", () => {
  const path = [
    { x: 5, y: 1, t: 2 }, { x: 1, y: 3, t: 0 }, { x: 4, y: 4, t: 3 }, { x: 2, y: 2, t: 1 },
  ];

  it("joins points in the order channel's order, not X order", () => {
    const out = run("Connected Scatter Plot", path, { x: { field: "x" }, y: { field: "y" }, order: { field: "t" } });
    expect(out.type).toBe("line");
    expect(out.data.series[0].values!.map((p) => [p.x, p.y])).toEqual([[1, 3], [2, 2], [5, 1], [4, 4]]);
  });

  it("without an order channel keeps the row order", () => {
    const out = run("Connected Scatter Plot", path, { x: { field: "x" }, y: { field: "y" } });
    expect(out.data.series[0].values!.map((p) => p.x)).toEqual([5, 1, 4, 2]);
  });

  it("one path per color, each in its own order", () => {
    const two = [
      { x: 1, y: 1, t: 1, g: "a" }, { x: 3, y: 3, t: 0, g: "a" },
      { x: 9, y: 9, t: 0, g: "b" }, { x: 8, y: 8, t: 1, g: "b" },
    ];
    const out = run("Connected Scatter Plot", two, { x: { field: "x" }, y: { field: "y" }, order: { field: "t" }, color: { field: "g" } });
    expect(out.data.series.map((s) => s.values!.map((p) => p.x))).toEqual([[3, 1], [9, 8]]);
  });
});

describe("Bubble Chart", () => {
  const bubbles = [{ x: 1, y: 2, s: 10 }, { x: 2, y: 3, s: 40 }, { x: 3, y: 1, s: 90 }];

  it("is a scatter plot, and says the size channel is not drawn", () => {
    const out = run("Bubble Chart", bubbles, { x: { field: "x" }, y: { field: "y" }, size: { field: "s" } });
    expect(out.type).toBe("scatter");
    const w = (out._warnings ?? []).find((x: any) => x.code === "chart-type-approximated");
    expect(w?.severity).toBe("info");
    expect(w?.message).toMatch(/size/i);
  });

  it("without a size channel there is nothing to report", () => {
    const out = run("Bubble Chart", bubbles, { x: { field: "x" }, y: { field: "y" } });
    expect(codes(out)).not.toContain("chart-type-approximated");
  });
});
