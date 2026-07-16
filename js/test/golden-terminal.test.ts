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
