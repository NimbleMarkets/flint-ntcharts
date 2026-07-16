import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { compileToNtSpec } from "../src/compile.js";

const fixtureDir = fileURLToPath(new URL("../../testdata/fixtures/", import.meta.url));

describe("compileToNtSpec envelope", () => {
  it("wraps the spec with warnings and size, stripping private keys", () => {
    const input = readFileSync(`${fixtureDir}bar-currency.json`, "utf8");
    const out = JSON.parse(compileToNtSpec(input));
    expect(Object.keys(out)).toEqual(["spec", "warnings", "size"]);
    expect(out.spec.type).toBe("bar");
    expect(Array.isArray(out.warnings)).toBe(true);
    expect(out.size).toEqual({ width: out.spec.width, height: out.spec.height });
    for (const key of Object.keys(out.spec)) expect(key.startsWith("_")).toBe(false);
  });
  it("stringifies compactly and deterministically", () => {
    const input = readFileSync(`${fixtureDir}scatter.json`, "utf8");
    const a = compileToNtSpec(input);
    const b = compileToNtSpec(input);
    expect(a).toBe(b);
    expect(a).not.toContain("\n");
  });
  it("returns an error envelope for an unknown chart type instead of throwing", () => {
    const input = JSON.stringify({
      data: { values: [{ a: 1 }] },
      chart_spec: { chartType: "Rose Chart", encodings: { x: { field: "a" } } },
    });
    const out = JSON.parse(compileToNtSpec(input));
    expect(Object.keys(out)).toEqual(["error"]);
    expect(out.error.message).toMatch(/Unknown chart type/);
  });
});
