import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { assembleNtcharts } from "../src/ntcharts/index.js";
import { expectGolden } from "./helpers.js";

function input(name = "combo"): any {
  return JSON.parse(readFileSync(new URL(`../../testdata/fixtures-terminal/${name}.json`, import.meta.url), "utf8"));
}
describe("Flint Combo Chart", () => {
  for (const name of ["combo", "combo-sparse", "combo-bars"]) {
    it(`emits ${name} with category-aligned series`, () => {
      const out = assembleNtcharts(input(name));
      expect(out.type).toBe("bar");
      expect(out.x_axis?.labels).toEqual(["Jan", "Feb", "Mar", "Apr", "May", "Jun"]);
      expect(out.data.series[0].type).toBe("bar");
      expect(out.y_axis?.min).toBeUndefined();
      expect(out._warnings).toBeUndefined();
      if (name === "combo-bars") {
        expect(out.data.series).toHaveLength(1);
        expect(out.y2_axis).toBeUndefined();
      } else {
        expect(out.data.series[1]).toMatchObject({type: "line", y_axis: "right", name: "margin"});
        expect(out.y2_axis?.format?.kind).toBe("percent");
        expect(out.y_axis?.format?.kind).toBe("currency");
      }
      if (name === "combo-sparse") {
        expect(out.data.series[0].values?.some(p => p.x === "Mar")).toBe(false);
        expect(out.data.series[1].values?.some(p => p.x === "Feb")).toBe(false);
        expect(out.data.series[0].values?.[0].y).toBe(-25);
      }
      expectGolden(name, out, "terminal");
    });
  }
  it("keeps positional slots in a sparse bars-only fallback", () => {
    const doc = input("combo-bars");
    doc.data.values[1].revenue = null;
    const out = assembleNtcharts(doc);
    expect(out.data.series[0].values?.slice(0, 3)).toEqual([
      {x: "Jan", y: 120}, {x: "Feb", y: 0}, {x: "Mar", y: 150},
    ]);
  });
  it("uses semantic month ordering for both measures", () => {
    const doc = input();
    doc.data.values.reverse();
    doc.semantic_types.month = {semanticType: "Month"};
    const out = assembleNtcharts(doc);
    expect(out.x_axis?.labels).toEqual(["Jan", "Feb", "Mar", "Apr", "May", "Jun"]);
    expect(out.data.series[1].values?.[0]).toEqual({x: "Jan", y: .2});
  });
  it("infers the line field and sums duplicate categories like Flint", () => {
    const doc = input();
    delete doc.chart_spec.chartProperties;
    doc.data.values.push({month: "Jan", revenue: 30, margin: .1});
    const out = assembleNtcharts(doc);
    expect(out.data.series[1].name).toBe("margin");
    expect(out.data.series[0].values?.[0].y).toBe(150);
    expect(out.data.series[1].values?.[0].y).toBeCloseTo(.3);
  });
  it("honors an explicit field instead of the first other numeric field", () => {
    const doc = input();
    doc.data.values.forEach((r: any) => r.temperature = 25);
    doc.chart_spec.chartProperties.lineField = "temperature";
    expect(assembleNtcharts(doc).data.series[1].name).toBe("temperature");
  });
  it("falls back to inference when the explicit field is absent, like Flint", () => {
    const doc = input(); doc.chart_spec.chartProperties.lineField = "absent";
    expect(assembleNtcharts(doc).data.series[1].name).toBe("margin");
  });
  it("keeps line gaps for invalid values and reports them", () => {
    const doc = input(); doc.data.values[0].margin = "bad";
    const out = assembleNtcharts(doc);
    expect(out.data.series[1].values).toHaveLength(5);
    expect(out._warnings?.[0].code).toBe("invalid-combo-value");
  });
  it("warns and stays linear for a log request", () => {
    const doc = input(); doc.chart_spec.chartProperties.logScale_y = true;
    const out = assembleNtcharts(doc);
    expect(out.y_axis?.scale).toBeUndefined();
    expect(out._warnings?.[0].code).toBe("log-scale-unsupported");
  });
  it("rejects unsupported aggregation instead of losing the measures", () => {
    const doc = input(); doc.chart_spec.encodings.y.aggregate = "average";
    expect(() => assembleNtcharts(doc)).toThrow(/pre-aggregate/);
  });
});
