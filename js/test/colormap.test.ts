import { describe, it, expect } from "vitest";
import { paletteForScheme, gradientForScheme } from "../src/ntcharts/colormap.js";

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
