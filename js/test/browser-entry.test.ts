import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { compileToNtSpec } from "../src/compile.js";
import "../src/entry-browser.js"; // registers the namespace on import

const fixtureDir = fileURLToPath(new URL("../../testdata/fixtures/", import.meta.url));

describe("browser entry registration", () => {
  it("registers boobaShim.flintchart with a working sync compile", async () => {
    const ns = (globalThis as any).boobaShim?.flintchart;
    expect(ns).toBeDefined();
    await ns.ready;
    const input = readFileSync(`${fixtureDir}bar-currency.json`, "utf8");
    expect(ns.compile(input)).toBe(compileToNtSpec(input)); // identical by construction
    expect(typeof ns.version).toBe("string");
  });
  it("returns an error envelope for bad input, never throws", () => {
    const ns = (globalThis as any).boobaShim.flintchart;
    const out = JSON.parse(ns.compile(`{"chart_spec":{"chartType":"Nope","encodings":{}},"data":{"values":[]}}`));
    expect(Object.keys(out)).toEqual(["error"]);
  });
});
