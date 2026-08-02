import { describe, it, expect } from "vitest";
import { assembleFixture, expectGolden } from "./helpers.js";

// Terminal-scale coverage: each fixture omits baseSize, so
// resolveBaseSizeShim falls back to DEFAULT_TERMINAL_BASE (64x20) and, with
// no canvasSize ceiling, deriveStretchCapsShim now uses TERMINAL_OPTIONS's
// maxStretch: 1 (Fix 1) -- the emitted width/height must never exceed the
// terminal base.

const TERMINAL_MAX_WIDTH = 64;
const TERMINAL_MAX_HEIGHT = 20;

function expectWithinTerminalBase(out: { width: number; height: number }) {
  expect(out.width).toBeLessThanOrEqual(TERMINAL_MAX_WIDTH);
  expect(out.height).toBeLessThanOrEqual(TERMINAL_MAX_HEIGHT);
}

describe("golden-terminal: bar-currency", () => {
  it("stays within the terminal base and keeps series data", () => {
    const out = assembleFixture("bar-currency", "terminal");
    expect(out.type).toBe("bar");
    expectWithinTerminalBase(out);
    expect(out.data.series.length).toBeGreaterThan(0);
    expectGolden("bar-currency", out, "terminal");
  });
});

describe("golden-terminal: stacked-bar", () => {
  it("stays within the terminal base and keeps series data", () => {
    const out = assembleFixture("stacked-bar", "terminal");
    expect(out.type).toBe("bar");
    expectWithinTerminalBase(out);
    expect(out.data.series.length).toBeGreaterThan(0);
    expectGolden("stacked-bar", out, "terminal");
  });
});

describe("golden-terminal: line-temporal", () => {
  it("stays within the terminal base and keeps series data", () => {
    const out = assembleFixture("line-temporal", "terminal");
    expect(out.type).toBe("timeseries");
    expectWithinTerminalBase(out);
    expect(out.data.series.length).toBeGreaterThan(0);
    expectGolden("line-temporal", out, "terminal");
  });

  it("pins a padded, fitted y-domain (zero excluded for price lines)", () => {
    const out = assembleFixture("line-temporal", "terminal");
    // Fixture y values: 104.2..119.3 (min 101.4? no — see below). Data:
    // [104.2, 108.9, 101.4, 115.7, 119.3, 112.8] → min 101.4, max 119.3,
    // span 17.9, pad 5% = 0.895.
    expect(out.y_axis?.min).toBeCloseTo(100.505, 3);
    expect(out.y_axis?.max).toBeCloseTo(120.195, 3);
  });
});

describe("golden-terminal: scatter", () => {
  it("stays within the terminal base and keeps series data", () => {
    const out = assembleFixture("scatter", "terminal");
    expect(out.type).toBe("scatter");
    expectWithinTerminalBase(out);
    expect(out.data.series.length).toBeGreaterThan(0);
    expectGolden("scatter", out, "terminal");
  });
});

describe("golden-terminal: heatmap", () => {
  it("stays within the terminal base and keeps heat cells", () => {
    const out = assembleFixture("heatmap", "terminal");
    expect(out.type).toBe("heatmap");
    expectWithinTerminalBase(out);
    expect(out.heat?.cells?.length).toBeGreaterThan(0);
    expectGolden("heatmap", out, "terminal");
  });
});

describe("golden-terminal: line-numeric", () => {
  it("emits a plain (non-timeseries) line spec with numeric ascending xs", () => {
    const out = assembleFixture("line-numeric", "terminal");
    expect(out.type).toBe("line"); // quantitative x, not temporal -> not "timeseries"
    expectWithinTerminalBase(out);
    const xs = out.data.series[0].values!.map((p: any) => p.x as number);
    expect(xs.length).toBeGreaterThan(0);
    for (const x of xs) expect(typeof x).toBe("number");
    expect(xs).toEqual([...xs].sort((a, b) => a - b));
    expectGolden("line-numeric", out, "terminal");
  });
});

describe("golden-terminal: candlestick", () => {
  it("emits time-sorted ohlc points with the up/down palette", () => {
    const out = assembleFixture("candlestick", "terminal");
    expect(out.type).toBe("ohlc");
    expectWithinTerminalBase(out);
    const pts = out.data.series[0].ohlc!;
    expect(pts).toHaveLength(5);
    const ts = pts.map((p: any) => p.t as number);
    expect(ts).toEqual([...ts].sort((a, b) => a - b));
    expect(pts[0]).toMatchObject({ o: 100, h: 108, l: 97, c: 105 });
    expect(out.theme?.palette).toEqual(["#26a69a", "#ef5350"]);
    expect(out.x_axis?.format?.kind).toBe("time");
    expectGolden("candlestick", out, "terminal");
  });
});

describe("golden-terminal: sparkline", () => {
  it("emits a single y-value series", () => {
    const out = assembleFixture("sparkline", "terminal");
    expect(out.type).toBe("sparkline");
    expectWithinTerminalBase(out);
    expect(out.data.series).toHaveLength(1);
    expect(out.data.series[0].values!.map((p: any) => p.y)).toEqual([3, 5, 2, 8, 6, 9, 4, 7]);
    expectGolden("sparkline", out, "terminal");
  });
});
