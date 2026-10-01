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

  it("logScale_y on a line chart emits a log Y axis and lets ntcharts pick decades", () => {
    const out = line({ logScale_y: true });
    expect(out.y_axis?.scale).toBe("log");
    // flint's zero baseline and fitted domain are linear-axis pins; a log axis has no zero
    expect(out.y_axis?.min).toBeUndefined();
    expect(out.y_axis?.max).toBeUndefined();
    expect(codes(out)).not.toContain("log-scale-unsupported");
  });

  it("logScale_y: false asks for nothing and warns about nothing", () => {
    const out = line({ logScale_y: false });
    expect(out.y_axis?.scale).toBeUndefined();
    expect(codes(out)).not.toContain("log-scale-unsupported");
  });
});

describe("assemble: logScale chart property", () => {
  const grow = Array.from({ length: 8 }, (_, i) => ({ x: i + 1, y: 10 ** (i / 2) }));
  const make = (chartType: string, extra: Record<string, unknown>, values: any[] = grow, encodings?: any) =>
    assembleNtcharts({
      data: { values },
      chart_spec: {
        chartType,
        encodings: encodings ?? { x: { field: "x" }, y: { field: "y" } },
        chartProperties: extra,
      },
    } as any);
  const warning = (out: any) => (out._warnings ?? []).find((w: any) => w.code === "log-scale-unsupported");

  it("scatter: both axes", () => {
    const out = make("Scatter Plot", { logScale_x: true, logScale_y: true });
    expect(out.x_axis?.scale).toBe("log");
    expect(out.y_axis?.scale).toBe("log");
    expect(warning(out)).toBeUndefined();
  });

  it("numeric line: X and Y", () => {
    const out = make("Line Chart", { logScale_x: true, logScale_y: true });
    expect(out.type).toBe("line");
    expect(out.x_axis?.scale).toBe("log");
    expect(out.y_axis?.scale).toBe("log");
  });

  it("time series: Y is honoured, X (time) is reported", () => {
    const dated = grow.map((r, i) => ({ d: `2026-0${i + 1}-01`, y: r.y }));
    const enc = { x: { field: "d" }, y: { field: "y" } };
    const y = make("Line Chart", { logScale_y: true }, dated, enc);
    expect(y.type).toBe("timeseries");
    expect(y.y_axis?.scale).toBe("log");
    const x = make("Line Chart", { logScale_x: true }, dated, enc);
    expect(x.x_axis?.scale).toBeUndefined();
    expect(warning(x)?.message).toMatch(/time/i);
    expect(warning(x)?.channel).toBe("x");
  });

  it("candlestick: Y", () => {
    const ohlc = [
      { d: "2026-01-05", o: 10, h: 14, l: 8, c: 12 },
      { d: "2026-01-06", o: 12, h: 90, l: 11, c: 80 },
      { d: "2026-01-07", o: 80, h: 400, l: 70, c: 300 },
    ];
    const out = make("Candlestick Chart", { logScale_y: true }, ohlc, {
      x: { field: "d" }, open: { field: "o" }, high: { field: "h" }, low: { field: "l" }, close: { field: "c" },
    });
    expect(out.type).toBe("ohlc");
    expect(out.y_axis?.scale).toBe("log");
  });

  it("data containing zero or negatives cannot sit on a log axis: reported, stays linear", () => {
    for (const bad of [0, -5]) {
      const values = grow.map((r, i) => (i === 3 ? { ...r, y: bad } : r));
      const out = make("Scatter Plot", { logScale_y: true }, values);
      expect(out.y_axis?.scale).toBeUndefined();
      expect(warning(out)?.message).toMatch(/zero or negative|positive/i);
    }
  });

  it("charts ntcharts cannot draw on a log axis are reported", () => {
    const cats = ["a", "b", "c"].map((k, i) => ({ k, v: 10 ** (i + 1) }));
    const bar = make("Bar Chart", { logScale_y: true }, cats, { x: { field: "k" }, y: { field: "v" } });
    expect(bar.y_axis?.scale).toBeUndefined();
    expect(warning(bar)?.message).toMatch(/bar/i);
    const spark = make("Sparkline", { logScale_y: true });
    expect(spark.y_axis?.scale).toBeUndefined();
    expect(warning(spark)).toBeDefined();
  });

  it("an axis that was not asked for is left alone", () => {
    const out = make("Scatter Plot", { logScale_y: true });
    expect(out.y_axis?.scale).toBe("log");
    expect(out.x_axis?.scale).toBeUndefined();
  });
});

// flint shrinks the plot of a chart whose X axis is continuous (a pixel-world
// aspect-ratio and mark-density calculation), emitting about 45% of the
// requested width. A terminal chart fills the cells it was given.
describe("assemble: charts fill the requested size", () => {
  const pts = Array.from({ length: 30 }, (_, i) => ({ x: i, y: Math.round(30 + 20 * Math.sin(i / 4)) }));
  const cats = ["a", "b", "c"].flatMap((a) => ["p", "q"].map((b) => ({ a, b, v: a.length + b.length })));
  const numericCols = ["09", "12", "15"].flatMap((h) => ["Mon", "Tue"].map((d) => ({ h, d, v: 1 })));

  const cases: [string, any, any][] = [
    ["line chart", "Line Chart", { x: { field: "x" }, y: { field: "y" } }],
    ["scatter plot", "Scatter Plot", { x: { field: "x" }, y: { field: "y" } }],
    ["sparkline", "Sparkline", { x: { field: "x" }, y: { field: "y" } }],
  ];

  for (const [name, chartType, encodings] of cases) {
    for (const [width, height] of [[80, 20], [120, 40], [30, 10]]) {
      it(`${name} at ${width}x${height} emits exactly that`, () => {
        const out = assembleNtcharts({
          data: { values: pts },
          chart_spec: { chartType, encodings, baseSize: { width, height } },
        } as any);
        expect([out.width, out.height]).toEqual([width, height]);
        expect([out._width, out._height]).toEqual([width, height]);
      });
    }
  }

  it("a heatmap whose column names look like numbers fills the width too", () => {
    const out = assembleNtcharts({
      data: { values: numericCols },
      chart_spec: {
        chartType: "Heatmap",
        encodings: { x: { field: "h" }, y: { field: "d" }, color: { field: "v" } },
        baseSize: { width: 60, height: 16 },
      },
    } as any);
    expect([out.width, out.height]).toEqual([60, 16]);
  });

  it("categorical charts are unchanged", () => {
    const heat = assembleNtcharts({
      data: { values: cats },
      chart_spec: {
        chartType: "Heatmap",
        encodings: { x: { field: "a" }, y: { field: "b" }, color: { field: "v" } },
        baseSize: { width: 80, height: 20 },
      },
    } as any);
    expect([heat.width, heat.height]).toEqual([80, 20]);
    const bar = assembleNtcharts({
      data: { values: [{ k: "a", v: 1 }, { k: "b", v: 2 }] },
      chart_spec: { chartType: "Bar Chart", encodings: { x: { field: "k" }, y: { field: "v" } }, baseSize: { width: 80, height: 20 } },
    } as any);
    expect([bar.width, bar.height]).toEqual([80, 20]);
  });

  it("never exceeds the request unless a canvasSize allows growth", () => {
    const out = assembleNtcharts({
      data: { values: pts },
      chart_spec: {
        chartType: "Line Chart",
        encodings: { x: { field: "x" }, y: { field: "y" } },
        baseSize: { width: 80, height: 20 },
        canvasSize: { width: 60, height: 20 },
      },
    } as any);
    expect(out.width).toBeLessThanOrEqual(60);
  });

  it("the 8x4 minimum still applies to a tiny request", () => {
    const out = assembleNtcharts({
      data: { values: pts },
      chart_spec: { chartType: "Line Chart", encodings: { x: { field: "x" }, y: { field: "y" } }, baseSize: { width: 5, height: 2 } },
    } as any);
    expect([out.width, out.height]).toEqual([8, 4]);
  });
});
