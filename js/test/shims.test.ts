import { describe, it, expect } from "vitest";
import { resolveBaseSizeShim, deriveStretchCapsShim, applyAggregationShim } from "../src/ntcharts/shims.js";

describe("shims", () => {
  it("resolveBaseSizeShim defaults and clamps to ceiling", () => {
    expect(resolveBaseSizeShim(undefined, undefined)).toEqual({ width: 64, height: 20 });
    expect(resolveBaseSizeShim({ width: 100, height: 30 }, { width: 80, height: 40 }))
      .toEqual({ width: 80, height: 30 });
  });
  it("deriveStretchCapsShim ratios ≥ 1", () => {
    expect(deriveStretchCapsShim({ width: 64, height: 20 }, { width: 128, height: 20 }))
      .toEqual({ maxStretchX: 2, maxStretchY: 1 });
    expect(deriveStretchCapsShim({ width: 64, height: 20 }, undefined))
      .toEqual({ maxStretchX: 1.5, maxStretchY: 1.5 });
  });
  it("applyAggregationShim sums grouped rows and passes through otherwise", () => {
    const rows = [
      { cat: "a", v: 1 }, { cat: "a", v: 2 }, { cat: "b", v: 5 },
    ];
    const encodings = { x: { field: "cat" }, y: { field: "v", aggregate: "sum" } };
    expect(applyAggregationShim(encodings as any, rows)).toEqual([
      { cat: "a", v: 3 }, { cat: "b", v: 5 },
    ]);
    const plain = { x: { field: "cat" }, y: { field: "v" } };
    expect(applyAggregationShim(plain as any, rows)).toEqual(rows);
  });
});
