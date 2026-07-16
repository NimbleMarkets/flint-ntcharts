import { describe, it, expect } from "vitest";
import { formatSpecToNt, d3TimeToGoLayout } from "../src/ntcharts/format.js";

const noWarn = () => {};

describe("formatSpecToNt", () => {
  it("maps currency prefix", () => {
    expect(formatSpecToNt({ pattern: ",.2f", prefix: "$" }, noWarn)).toEqual({
      kind: "currency", currency: "$", precision: 2,
    });
  });
  it("maps d3 percent patterns", () => {
    expect(formatSpecToNt({ pattern: ".0~%" }, noWarn)).toEqual({ kind: "percent", precision: 0 });
    expect(formatSpecToNt({ pattern: ".1%" }, noWarn)).toEqual({ kind: "percent", precision: 1 });
  });
  it("maps abbreviate to si", () => {
    expect(formatSpecToNt({ abbreviate: true }, noWarn)).toEqual({ kind: "si", precision: 1 });
  });
  it("maps plain decimal patterns to number+precision", () => {
    expect(formatSpecToNt({ pattern: ",.3f" }, noWarn)).toEqual({ kind: "number", precision: 3 });
    expect(formatSpecToNt({ pattern: ",d" }, noWarn)).toEqual({ kind: "number", precision: 0 });
  });
  it("drops suffix with a warning and returns undefined for empty", () => {
    const warns: unknown[] = [];
    expect(formatSpecToNt({ suffix: "°C" }, (w) => warns.push(w))).toBeUndefined();
    expect(warns).toHaveLength(1);
    expect(formatSpecToNt({}, noWarn)).toBeUndefined();
    expect(formatSpecToNt(undefined, noWarn)).toBeUndefined();
  });
});

describe("d3TimeToGoLayout", () => {
  it("translates common tokens", () => {
    expect(d3TimeToGoLayout("%Y-%m-%d", noWarn)).toBe("2006-01-02");
    expect(d3TimeToGoLayout("%b %d", noWarn)).toBe("Jan 02");
    expect(d3TimeToGoLayout("%H:%M", noWarn)).toBe("15:04");
    expect(d3TimeToGoLayout("%Y", noWarn)).toBe("2006");
  });
  it("warns on untranslatable tokens and passes them through", () => {
    const warns: unknown[] = [];
    expect(d3TimeToGoLayout("%Y w%U", (w) => warns.push(w))).toBe("2006 w%U");
    expect(warns).toHaveLength(1);
  });
});
