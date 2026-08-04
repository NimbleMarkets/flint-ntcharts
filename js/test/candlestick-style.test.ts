import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { assembleNtcharts } from "../src/ntcharts/index.js";

const fixturePath = fileURLToPath(
  new URL("../../testdata/fixtures-terminal/candlestick.json", import.meta.url));

function candlestickInput(candleStyle?: string) {
  const input = JSON.parse(readFileSync(fixturePath, "utf8"));
  if (candleStyle !== undefined) {
    input.chart_spec.chartProperties = { candleStyle };
  }
  return input;
}

describe("candlestick candleStyle passthrough", () => {
  it("emits options.candle_style for 'block'", () => {
    const out = assembleNtcharts(candlestickInput("block"));
    expect(out.options?.candle_style).toBe("block");
  });
  it("emits options.candle_style for 'line'", () => {
    const out = assembleNtcharts(candlestickInput("line"));
    expect(out.options?.candle_style).toBe("line");
  });
  it("ignores unknown values and absence", () => {
    expect(assembleNtcharts(candlestickInput("fancy")).options?.candle_style).toBeUndefined();
    expect(assembleNtcharts(candlestickInput()).options?.candle_style).toBeUndefined();
  });
});
