import { describe, it, expect } from "vitest";
import { assembleNtcharts } from "../src/ntcharts/index.js";
import { warningKey } from "./helpers.js";

function manyCategories(n: number) {
  return Array.from({ length: n }, (_, i) => ({ product: `item-${String(i).padStart(2, "0")}`, revenue: 10 + i }));
}

describe("assemble: overflow warnings", () => {
  it("reports each overflow once, not once as a warning and once as a truncation", () => {
    // flint keeps at least 60 discrete values before overflowing, whatever
    // the width, so the fixture needs well over that to trigger it.
    const out = assembleNtcharts({
      data: { values: manyCategories(120) },
      chart_spec: {
        chartType: "Bar Chart",
        encodings: { x: { field: "product" }, y: { field: "revenue" } },
        baseSize: { width: 20, height: 8 },
      },
    });
    const warnings = out._warnings ?? [];
    const overflow = warnings.filter((w) => w.code === "overflow");
    expect(overflow.length, "fixture must actually overflow").toBeGreaterThan(0);
    const distinct = new Set(warnings.map(warningKey));
    expect(distinct.size).toBe(warnings.length);
    expect(overflow.filter((w) => w.channel === "x")).toHaveLength(1);
  });
});

describe("assemble: title and subtitle", () => {
  const base = {
    data: { values: manyCategories(3) },
    chart_spec: {
      chartType: "Bar Chart",
      encodings: { x: { field: "product" }, y: { field: "revenue" } },
    },
  };

  it("passes chart_spec.title and subtitle through to the spec", () => {
    const out = assembleNtcharts({
      ...base,
      chart_spec: { ...base.chart_spec, title: "Revenue by product", subtitle: "FY26" },
    });
    expect(out.title).toBe("Revenue by product");
    expect(out.subtitle).toBe("FY26");
  });

  it("omits the keys entirely when the input has none", () => {
    const out = assembleNtcharts(base);
    expect("title" in out).toBe(false);
    expect("subtitle" in out).toBe(false);
  });
});

// The ntcharts bar model needs two cells per bar (the bar and its gap); with
// more bars than that it computes a bar width of 0 and draws only the axis.
// The compiler must therefore never emit more categories than fit.
describe("assemble: bar charts fit the renderer", () => {
  const capacity = (cells: number) => Math.floor((cells + 1) / 2);

  for (const width of [8, 12, 20, 40, 64]) {
    it(`vertical bars at width ${width}: categories fit, overflow reported once`, () => {
      const out = assembleNtcharts({
        data: { values: manyCategories(100) },
        chart_spec: {
          chartType: "Bar Chart",
          encodings: { x: { field: "product" }, y: { field: "revenue" } },
          baseSize: { width, height: 10 },
        },
      });
      const labels = out.x_axis?.labels ?? [];
      expect(labels.length).toBeGreaterThan(0);
      expect(labels.length, `${labels.length} bars cannot fit in ${out.width} cells`).toBeLessThanOrEqual(capacity(out.width));
      const overflow = (out._warnings ?? []).filter((w) => w.code === "overflow");
      expect(overflow).toHaveLength(1);
    });
  }

  it("horizontal bars: categories fit the height", () => {
    const out = assembleNtcharts({
      data: { values: manyCategories(100) },
      chart_spec: {
        chartType: "Bar Chart",
        encodings: { x: { field: "revenue" }, y: { field: "product" } },
        baseSize: { width: 40, height: 12 },
      },
    });
    expect(out.options?.orientation).toBe("horizontal");
    const labels = out.x_axis?.labels ?? [];
    expect(labels.length).toBeGreaterThan(0);
    expect(labels.length).toBeLessThanOrEqual(capacity(out.height));
  });

  it("a chart whose categories already fit is not truncated", () => {
    const out = assembleNtcharts({
      data: { values: manyCategories(5) },
      chart_spec: {
        chartType: "Bar Chart",
        encodings: { x: { field: "product" }, y: { field: "revenue" } },
        baseSize: { width: 20, height: 8 },
      },
    });
    expect(out.x_axis?.labels).toHaveLength(5);
    expect((out._warnings ?? []).filter((w) => w.code === "overflow")).toHaveLength(0);
  });
});

describe("assemble: includeZero and logScale chart properties", () => {
  const prices = [
    { d: "2026-01-01", price: 104.2 }, { d: "2026-02-01", price: 108.9 },
    { d: "2026-03-01", price: 101.4 }, { d: "2026-04-01", price: 119.3 },
  ];
  const line = (chartProperties?: Record<string, unknown>, semantic = "Price") => assembleNtcharts({
    data: { values: prices },
    semantic_types: { price: semantic },
    chart_spec: {
      chartType: "Line Chart",
      encodings: { x: { field: "d" }, y: { field: "price" } },
      ...(chartProperties ? { chartProperties } : {}),
    },
  });
  const codes = (out: { _warnings?: { code?: string }[] }) => (out._warnings ?? []).map((w) => w.code);

  it("baseline: a Price line keeps flint's zero baseline", () => {
    expect(line().y_axis?.min).toBe(0);
  });

  it("includeZero_y: false fits the Y domain to the data", () => {
    const out = line({ includeZero_y: false });
    expect(out.y_axis?.min).toBeGreaterThan(90);
    expect(out.y_axis?.min).toBeLessThan(101.4);
    expect(out.y_axis?.max).toBeGreaterThan(119.3);
    expect(codes(out)).not.toContain("chart-property-unsupported");
  });

  it("includeZero_y: true forces a zero baseline flint would not choose", () => {
    expect(line(undefined, "Temperature").y_axis?.min).toBeGreaterThan(90);
    expect(line({ includeZero_y: true }, "Temperature").y_axis?.min).toBe(0);
  });

  it("includeZero_y on a bar chart is reported, not silently dropped", () => {
    const out = assembleNtcharts({
      data: { values: manyCategories(3) },
      chart_spec: {
        chartType: "Bar Chart",
        encodings: { x: { field: "product" }, y: { field: "revenue" } },
        chartProperties: { includeZero_y: false },
      },
    });
    expect(out.y_axis?.min).toBe(0);
    expect(codes(out)).toContain("chart-property-unsupported");
  });

  it("includeZero_x is reported: the terminal X axis is never pinned", () => {
    expect(codes(line({ includeZero_x: true }))).toContain("chart-property-unsupported");
  });

  it("logScale_y is reported and the chart stays linear", () => {
    const out = line({ logScale_y: true });
    const w = (out._warnings ?? []).find((x) => x.code === "log-scale-unsupported");
    expect(w, "expected a log-scale-unsupported warning").toBeDefined();
    expect(w!.channel).toBe("y");
  });

  it("logScale_y: false asks for nothing and warns about nothing", () => {
    expect(codes(line({ logScale_y: false }))).not.toContain("log-scale-unsupported");
  });
});
