import { describe, it, expect } from "vitest";
import { assembleFixture, expectGolden } from "./helpers.js";
import { assembleNtcharts } from "../src/ntcharts/index.js";

describe("golden: bar-currency", () => {
  it("emits a stacked=false single-series bar spec with currency Y format", () => {
    const out = assembleFixture("bar-currency");
    expect(out.type).toBe("bar");
    expect(out.data.series).toHaveLength(1);
    expect(out.x_axis?.labels).toContain("Widgets");
    expect(out.y_axis?.format?.kind).toBe("currency");
    expect(out.y_axis?.min).toBe(0); // bar zero decision
    expect(out.width).toBeGreaterThanOrEqual(8);
    expect(out.height).toBeGreaterThanOrEqual(4);
    expectGolden("bar-currency", out);
  });
});

describe("bar: group channel splits series correctly", () => {
  it("gives each group-split series its own values", () => {
    const out = assembleNtcharts({
      data: { values: [
        { product: "A", revenue: 10, region: "east" },
        { product: "B", revenue: 20, region: "east" },
        { product: "A", revenue: 30, region: "west" },
        { product: "B", revenue: 40, region: "west" },
      ]},
      chart_spec: {
        chartType: "Bar Chart",
        encodings: { x: { field: "product" }, y: { field: "revenue" }, group: { field: "region" } },
        baseSize: { width: 40, height: 12 },
      },
    } as any);
    expect(out.data.series).toHaveLength(2);
    const east = out.data.series.find((s: any) => s.name === "east")!;
    const west = out.data.series.find((s: any) => s.name === "west")!;
    expect(east.values!.map((p: any) => p.y)).toEqual([10, 20]);
    expect(west.values!.map((p: any) => p.y)).toEqual([30, 40]);
  });
});

describe("golden: stacked-bar", () => {
  it("emits a multi-series stacked bar spec", () => {
    const out = assembleFixture("stacked-bar");
    expect(out.type).toBe("bar");
    expect(out.options?.stacked).toBe(true);
    expect(out.data.series.length).toBe(2);
    expect(out.data.series.map((s: any) => s.name).sort()).toEqual(["APAC", "EMEA"]);
    expect(out.data.series[0].color).toMatch(/^#[0-9a-f]{6}$/);
    expectGolden("stacked-bar", out);
  });
});

describe("golden: line-temporal", () => {
  it("emits a timeseries spec with ms X values and a time format", () => {
    const out = assembleFixture("line-temporal");
    expect(out.type).toBe("timeseries");
    const xs = out.data.series[0].values!.map((p: any) => p.x);
    for (const x of xs) expect(typeof x).toBe("number"); // ms since epoch
    expect(xs[0]).toBeGreaterThan(1.7e12);
    expect(out.x_axis?.format?.kind).toBe("time");
    expect(out.x_axis?.format?.layout).not.toMatch(/%/); // fully translated Go layout
    expectGolden("line-temporal", out);
  });
});

describe("line: robustness", () => {
  const base = {
    chart_spec: {
      chartType: "Line Chart",
      encodings: { x: { field: "date" }, y: { field: "value" } },
      baseSize: { width: 40, height: 12 },
    },
  };
  it("sorts points by x even when rows arrive out of order", () => {
    const out = assembleNtcharts({
      ...base,
      data: { values: [
        { date: "2026-03-01", value: 3 },
        { date: "2026-01-01", value: 1 },
        { date: "2026-02-01", value: 2 },
      ]},
    } as any);
    const xs = out.data.series[0].values!.map((p: any) => p.x as number);
    expect(xs).toEqual([...xs].sort((a, b) => a - b));
    expect(out.data.series[0].values!.map((p: any) => p.y)).toEqual([1, 2, 3]);
  });
  // NOTE: a bare { field: "date" } encoding (as in `base`) won't reproduce the
  // NaN path end-to-end here: flint-chart's inferVisCategory/isDate type
  // inference is all-or-nothing (see semantic-types.ts) — one unparsable
  // string flips the *whole column* to "nominal", so convertTemporalData
  // never touches it and channelSemantics.x.type !== "temporal" (toMs is
  // never called at all; Number("not-a-date") short-circuits to NaN/null
  // through an entirely different, already-existing branch). To exercise
  // the actual toMs()-returns-NaN / drop+warn path end-to-end, we force
  // temporal typing explicitly via `encodings.x.type` + a matching
  // `semantic_types` entry, which flint-chart honors unconditionally
  // (resolve-semantics.ts: `if (encoding.type) resolvedType = encoding.type`)
  // and which makes convertTemporalData leave the unparsable cell as a raw
  // string. That is the shape production callers use for known-temporal
  // fields, so this is still a realistic end-to-end exercise, not a unit seam.
  it("drops unparsable temporal points with a warning instead of plotting epoch", () => {
    const out = assembleNtcharts({
      chart_spec: {
        chartType: "Line Chart",
        encodings: { x: { field: "date", type: "temporal" }, y: { field: "value" } },
        baseSize: { width: 40, height: 12 },
      },
      semantic_types: { date: "temporal" },
      data: { values: [
        { date: "2026-01-01", value: 1 },
        { date: "not-a-date", value: 99 },
        { date: "2026-02-01", value: 2 },
      ]},
    } as any);
    expect(out.data.series[0].values).toHaveLength(2);
    expect(out.data.series[0].values!.every((p: any) => (p.x as number) > 1.7e12)).toBe(true);
    expect((out._warnings ?? []).some((w: any) => w.code === "invalid-temporal-x")).toBe(true);
  });
});
