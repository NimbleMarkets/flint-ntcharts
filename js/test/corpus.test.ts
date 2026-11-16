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

// Upstream cases the terminal declines on purpose, each with the reason it
// gives: a chart it would otherwise draw empty, collapsed or merged. Anything
// else that errors is a regression.
const DECLINED: Record<string, string> = {
  "Bar Chart :: N(5)×N(5) grid": "needs one quantitative axis",
  "Bar Chart :: T×N(5) (25 pts)": "needs one quantitative axis",
  "Bar Chart :: N(5)×T (25 pts)": "needs one quantitative axis",
  "Stacked Bar Chart :: N(5)×N(5) +color(N,3) grid": "needs one quantitative axis",
  "Grouped Bar Chart :: N(5)×N(5) +color(N,3) grid": "needs one quantitative axis",
  "Line Chart :: Q×O(5) (5 pts)": "measures along y",
  "Line Chart :: Q×O(12) +color(N,4) (48 pts)": "measures along y",
  "Line Chart :: Q×O(30) (30 pts)": "measures along y",
  "Scatter Plot :: N(5)×N(6) +size(Q) grid": "needs a measured axis",
  "Scatter Plot :: N(5)×N(4) +color(Q) grid": "needs a measured axis",
  "Scatter Plot :: N(15)×N(12) +size(Q) grid": "needs a measured axis",
  "Candlestick Chart :: Multi-stock facet (3 tickers)": "draws one instrument",
};

describe("upstream test-data corpus", () => {
  it("covers every registered chart type with at least one case", () => {
    const types = new Set(corpus.map((c) => c.chartType));
    // Bubble Chart is registered but upstream has no test generator for it;
    // templates-group1.test.ts covers it by hand.
    expect([...types].sort()).toEqual(
      ["Area Chart", "Bar Chart", "Calendar Heatmap", "Candlestick Chart", "Combo Chart", "Connected Scatter Plot", "Density Plot", "ECDF Plot",
       "Grouped Bar Chart", "Heatmap", "Histogram", "Line Chart", "Lollipop Chart", "Pyramid Chart", "Range Area Chart", "Ranged Dot Plot",
       "Regression", "Scatter Plot", "Slope Chart", "Sparkline", "Stacked Bar Chart", "Streamgraph", "Strip Plot"],
    );
    expect(corpus.length).toBeGreaterThan(50);
    // Every declined case is still in the corpus: a renamed upstream case
    // must not silently drop out of this check.
    const ids = new Set(corpus.map((c) => c.id));
    for (const id of Object.keys(DECLINED)) expect(ids.has(id), id).toBe(true);
  });

  for (const variant of ["pixel", "terminal"] as const) {
    describe(variant, () => {
      for (const c of corpus) {
        it(c.id, () => {
          const input = structuredClone(c.input);
          if (variant === "terminal") input.chart_spec.baseSize = { width: 64, height: 20 };
          const out = JSON.parse(compileToNtSpec(JSON.stringify(input)));
          if (c.id in DECLINED) {
            expect(out.error?.message, "declined with its reason").toContain(DECLINED[c.id]);
            return;
          }
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
