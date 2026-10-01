import { describe, it, expect } from "vitest";
import { compileToNtSpec } from "../src/compile.js";

const grouped = (extra: Record<string, unknown> = {}) =>
  JSON.stringify({
    data: { values: [{ c: "a", n: 3, g: "x" }, { c: "b", n: 5, g: "x" }, { c: "a", n: 2, g: "y" }, { c: "b", n: 4, g: "y" }] },
    chart_spec: {
      chartType: "Grouped Bar Chart",
      encodings: { x: { field: "c" }, y: { field: "n" }, group: { field: "g" } },
      baseSize: { width: 80, height: 24 },
    },
    ...extra,
  });

const compile = (json: string) => JSON.parse(compileToNtSpec(json));

describe("renderer: raster", () => {
  it("returns an ECharts option in place of a spec, with the requested size", () => {
    const env = compile(grouped({ renderer: "raster" }));
    expect(env.error).toBeUndefined();
    expect(env.spec).toBeUndefined();
    expect(Array.isArray(env.echarts.series)).toBe(true);
    expect(env.echarts.series[0].type).toBe("bar");
    expect(env.warnings).toEqual([]);
    expect(env.size).toEqual({ width: 80, height: 24 });
  });

  it("strips the private scratch keys from the option", () => {
    const env = compile(grouped({ renderer: "raster" }));
    expect(Object.keys(env.echarts).filter((k) => k.startsWith("_"))).toEqual([]);
  });

  it("draws chart types the text renderer also supports", () => {
    const env = compile(JSON.stringify({
      renderer: "raster",
      data: { values: [{ m: "Jan", r: 1 }, { m: "Feb", r: 2 }] },
      chart_spec: { chartType: "Bar Chart", encodings: { x: { field: "m" }, y: { field: "r" } }, baseSize: { width: 60, height: 16 } },
    }));
    expect(env.echarts.series[0].type).toBe("bar");
  });

  it("falls back to the option's own size when none is requested", () => {
    const input = JSON.parse(grouped({ renderer: "raster" }));
    delete input.chart_spec.baseSize;
    const env = compile(JSON.stringify(input));
    // whole cells: the layout's own size is fractional, the envelope's is not
    for (const n of [env.size.width, env.size.height]) {
      expect(Number.isInteger(n)).toBe(true);
      expect(n).toBeGreaterThan(0);
    }
  });

  it("reports a chart type flint's ECharts backend does not know as an error", () => {
    const env = compile(JSON.stringify({
      renderer: "raster",
      data: { values: [{ a: 1 }] },
      chart_spec: { chartType: "No Such Chart", encodings: {} },
    }));
    expect(env.error.message).toMatch(/No Such Chart/);
  });

  it("rejects a renderer it does not know", () => {
    const env = compile(grouped({ renderer: "pixels" }));
    expect(env.error.message).toMatch(/renderer/);
    expect(env.error.message).toMatch(/raster/);
  });

  it("treats renderer: text as the default text path", () => {
    const env = compile(JSON.stringify({
      renderer: "text",
      data: { values: [{ m: "Jan", r: 1 }, { m: "Feb", r: 2 }] },
      chart_spec: { chartType: "Bar Chart", encodings: { x: { field: "m" }, y: { field: "r" } } },
    }));
    expect(env.spec.type).toBe("bar");
    expect(env.echarts).toBeUndefined();
  });
});

describe("the text path, without a renderer", () => {
  it("is unchanged: a spec envelope", () => {
    const env = compile(JSON.stringify({
      data: { values: [{ m: "Jan", r: 1 }, { m: "Feb", r: 2 }] },
      chart_spec: { chartType: "Bar Chart", encodings: { x: { field: "m" }, y: { field: "r" } } },
    }));
    expect(env.spec.type).toBe("bar");
    expect(env.echarts).toBeUndefined();
  });

  it("points an unsupported chart type at the raster renderer", () => {
    const env = compile(grouped());
    expect(env.error.message).toMatch(/Unknown chart type "Grouped Bar Chart"/);
    expect(env.error.message).toContain('"renderer": "raster"');
  });
});
