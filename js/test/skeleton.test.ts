import { describe, it, expect } from "vitest";
import { assembleNtcharts } from "../src/ntcharts/index.js";

describe("assembleNtcharts skeleton", () => {
  it("rejects unknown chart types with the supported list", () => {
    expect(() =>
      assembleNtcharts({
        data: { values: [{ a: 1, b: 2 }] },
        chart_spec: { chartType: "Rose Chart", encodings: { x: { field: "a" }, y: { field: "b" } } },
      } as any),
    ).toThrow(/Unknown chart type "Rose Chart"/);
  });

  // Plan-mandated behavior (README: "Grouped Bar Chart is intentionally
  // unsupported"): ntcharts' terminal barchart.Model can't render
  // side-by-side groups, so this chart type is deliberately absent from the
  // template registry rather than silently rendered stacked or grouped.
  it("rejects \"Grouped Bar Chart\" as an unknown/unsupported chart type", () => {
    expect(() =>
      assembleNtcharts({
        data: { values: [{ product: "A", revenue: 10, region: "east" }] },
        chart_spec: {
          chartType: "Grouped Bar Chart",
          encodings: { x: { field: "product" }, y: { field: "revenue" }, group: { field: "region" } },
        },
      } as any),
    ).toThrow(/Unknown chart type "Grouped Bar Chart"/);
  });
});
