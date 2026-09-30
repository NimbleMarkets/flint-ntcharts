import { describe, it, expect } from "vitest";
import { SemanticTypes, getRecommendedColorScheme } from "flint-chart/core";
import { paletteForScheme, gradientForScheme, isKnownScheme } from "../src/ntcharts/colormap.js";

// Every scheme name flint-chart can hand a backend, found by driving its
// recommender across the whole semantic-type registry, every encoding type,
// both cardinality branches and both colour hints. An upstream rename or a
// new scheme fails here instead of silently falling back to viridis /
// tableau10 at render time (0.5.1 added `blueorange` this way).
function upstreamSchemes(): Map<string, string> {
  const found = new Map<string, string>(); // scheme -> categorical|sequential|diverging
  const semanticTypes = [...Object.keys(SemanticTypes), undefined];
  for (const st of semanticTypes) {
    for (const enc of ["nominal", "ordinal", "quantitative", "temporal", undefined]) {
      for (const count of [3, 12]) {
        for (const hint of [undefined, { type: "diverging" }, { type: "sequential" }]) {
          for (const field of ["a", "revenue", "xx", "q1"]) {
            const r = getRecommendedColorScheme(st as any, enc as any, count, field, [], hint as any);
            if (r?.scheme) found.set(r.scheme, r.type);
          }
        }
      }
    }
  }
  return found;
}

describe("colormap covers every scheme upstream can emit", () => {
  const schemes = upstreamSchemes();
  it("probe found the known baseline", () => {
    for (const s of ["tableau10", "viridis", "blueorange", "redblue"]) expect(schemes.has(s), s).toBe(true);
  });
  for (const [scheme, kind] of schemes) {
    it(`${scheme} (${kind})`, () => {
      expect(isKnownScheme(scheme), `add "${scheme}" to colormap.ts`).toBe(true);
      const stops = kind === "categorical" ? paletteForScheme(scheme) : gradientForScheme(scheme);
      for (const stop of stops) expect(stop).toMatch(/^#[0-9a-f]{6}$/);
    });
  }
});

describe("colormap", () => {
  it("returns tableau10 for the default categorical scheme", () => {
    const p = paletteForScheme("tableau10");
    expect(p[0]).toBe("#4e79a7");
    expect(p).toHaveLength(10);
  });
  it("falls back to tableau10 for unknown categorical schemes", () => {
    expect(paletteForScheme("no-such-scheme")).toEqual(paletteForScheme("tableau10"));
    expect(paletteForScheme(undefined)).toEqual(paletteForScheme("tableau10"));
  });
  it("returns ordered hex stops for sequential schemes", () => {
    const g = gradientForScheme("viridis");
    expect(g.length).toBeGreaterThanOrEqual(5);
    for (const stop of g) expect(stop).toMatch(/^#[0-9a-f]{6}$/);
  });
  it("falls back to viridis for unknown sequential schemes", () => {
    expect(gradientForScheme("mystery")).toEqual(gradientForScheme("viridis"));
  });
});
