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
});
