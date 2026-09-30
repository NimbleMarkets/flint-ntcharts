import { describe, it, expect } from "vitest";
import { assembleNtcharts } from "../src/ntcharts/index.js";
import { warningKey } from "./helpers.js";

function manyCategories(n: number) {
  return Array.from({ length: n }, (_, i) => ({ product: `item-${String(i).padStart(2, "0")}`, revenue: 10 + i }));
}

describe("assemble: overflow warnings", () => {
  it("reports each overflow once, not once as a warning and once as a truncation", () => {
    // flint keeps at least 60 discrete values before overflowing, whatever
    // the width, so the fixture needs well over that to trigger it.
    const out = assembleNtcharts({
      data: { values: manyCategories(120) },
      chart_spec: {
        chartType: "Bar Chart",
        encodings: { x: { field: "product" }, y: { field: "revenue" } },
        baseSize: { width: 20, height: 8 },
      },
    });
    const warnings = out._warnings ?? [];
    const overflow = warnings.filter((w) => w.code === "overflow");
    expect(overflow.length, "fixture must actually overflow").toBeGreaterThan(0);
    const distinct = new Set(warnings.map(warningKey));
    expect(distinct.size).toBe(warnings.length);
    expect(overflow.filter((w) => w.channel === "x")).toHaveLength(1);
  });
});

describe("assemble: title and subtitle", () => {
  const base = {
    data: { values: manyCategories(3) },
    chart_spec: {
      chartType: "Bar Chart",
      encodings: { x: { field: "product" }, y: { field: "revenue" } },
    },
  };

  it("passes chart_spec.title and subtitle through to the spec", () => {
    const out = assembleNtcharts({
      ...base,
      chart_spec: { ...base.chart_spec, title: "Revenue by product", subtitle: "FY26" },
    });
    expect(out.title).toBe("Revenue by product");
    expect(out.subtitle).toBe("FY26");
  });

  it("omits the keys entirely when the input has none", () => {
    const out = assembleNtcharts(base);
    expect("title" in out).toBe(false);
    expect("subtitle" in out).toBe(false);
  });
});
