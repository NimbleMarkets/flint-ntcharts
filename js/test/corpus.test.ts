import { describe, it, expect } from "vitest";
import { compileToNtSpec } from "../src/compile.js";
import { supportedCorpus } from "./corpus-helpers.js";
import { warningKey } from "./helpers.js";

// Semantic re-vendor guard: every upstream test case for a chart type we
// register must compile to a well-formed envelope, at pixel scale and at the
// terminal default. A flint-chart upgrade that makes any of them throw, or
// emit an error envelope, fails here before it reaches the wasm build.

const NT_TYPES = new Set(["bar", "line", "timeseries", "streamline", "sparkline", "heatmap", "ohlc", "scatter", "canvas"]);

const corpus = supportedCorpus();

describe("upstream test-data corpus", () => {
  it("covers every registered chart type with at least one case", () => {
    const types = new Set(corpus.map((c) => c.chartType));
    // Bubble Chart is registered but upstream has no test generator for it;
    // templates-group1.test.ts covers it by hand.
    expect([...types].sort()).toEqual(
      ["Area Chart", "Bar Chart", "Candlestick Chart", "Connected Scatter Plot", "ECDF Plot", "Heatmap", "Histogram", "Line Chart", "Lollipop Chart", "Scatter Plot", "Sparkline", "Stacked Bar Chart"],
    );
    expect(corpus.length).toBeGreaterThan(50);
  });

  for (const variant of ["pixel", "terminal"] as const) {
    describe(variant, () => {
      for (const c of corpus) {
        it(c.id, () => {
          const input = structuredClone(c.input);
          if (variant === "terminal") input.chart_spec.baseSize = { width: 64, height: 20 };
          const out = JSON.parse(compileToNtSpec(JSON.stringify(input)));
          expect(out.error, `error envelope: ${out.error?.message}`).toBeUndefined();
          expect(NT_TYPES.has(out.spec.type), `unknown spec.type ${out.spec.type}`).toBe(true);
          expect(out.spec.width).toBeGreaterThanOrEqual(8);
          expect(out.spec.height).toBeGreaterThanOrEqual(4);
          if (variant === "terminal") {
            expect(out.spec.width).toBeLessThanOrEqual(64);
            expect(out.spec.height).toBeLessThanOrEqual(20);
          }
          expect(Array.isArray(out.warnings)).toBe(true);
          // A warning that appears twice is a backend bug (see assemble.ts:
          // overflow warnings vs truncations), not two problems.
          const distinct = new Set(out.warnings.map(warningKey));
          expect(distinct.size, "duplicate warnings").toBe(out.warnings.length);
        });
      }
    });
  }
});
