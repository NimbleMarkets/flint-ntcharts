import { describe, it, expect } from "vitest";
import { assembleNtcharts } from "../src/ntcharts/index.js";
import { compileToNtSpec } from "../src/compile.js";
import { edgeLabels } from "../src/ntcharts/templates/histogram.js";
import { resample } from "../src/ntcharts/templates/sparkline.js";
import { kde } from "../src/ntcharts/templates/density.js";
import { fitCurve } from "../src/ntcharts/templates/regression.js";
import { regularDates } from "../src/ntcharts/cadence.js";
import { barSlots, centerSlots, planTicks, tickKind } from "../src/ntcharts/ticks.js";
import { categoryLabels } from "../src/ntcharts/bands.js";

// What the terminal draws must say what Flint's other backends say about the
// same input, or say plainly why it cannot. Each case here was drawn empty,
// clipped, merged or run together before.

const T = { width: 64, height: 20 };
const chart = (chartType: string, values: Record<string, unknown>[], encodings: Record<string, unknown>, semantic_types: Record<string, string>, extra: Record<string, unknown> = {}) =>
  ({ data: { values }, semantic_types, chart_spec: { chartType, encodings, baseSize: T, ...extra } }) as any;
const codes = (out: any) => (out._warnings ?? []).map((w: any) => w.code);
const errorOf = (input: unknown) => JSON.parse(compileToNtSpec(JSON.stringify(input))).error?.message as string | undefined;

describe("bars", () => {
  it("keep bars below zero: the value axis is not pinned at 0 when values go negative", () => {
    const out = assembleNtcharts(chart("Bar Chart",
      [{ d: "1900s", v: -0.16 }, { d: "1950s", v: 0.02 }, { d: "2000s", v: 0.4 }],
      { x: { field: "d" }, y: { field: "v" } }, { d: "Category", v: "Temperature" }));
    expect(out.y_axis?.min).toBeUndefined();
    expect(out.data.series[0].values!.map((p) => p.y)).toEqual([-0.16, 0.02, 0.4]);
  });

  it("still pin zero for all-positive bars", () => {
    const out = assembleNtcharts(chart("Bar Chart", [{ c: "a", v: 3 }, { c: "b", v: 5 }],
      { x: { field: "c" }, y: { field: "v" } }, { c: "Category", v: "Quantity" }));
    expect(out.y_axis?.min).toBe(0);
  });

  it("combine several rows on one bar by the measure's rule, and say so", () => {
    const rows = [{ c: "a", v: 2 }, { c: "a", v: 4 }, { c: "b", v: 5 }];
    const summed = assembleNtcharts(chart("Bar Chart", rows, { x: { field: "c" }, y: { field: "v" } }, { c: "Category", v: "Quantity" }));
    expect(summed.data.series[0].values!.map((p) => p.y)).toEqual([6, 5]);
    expect(codes(summed)).toContain("rows-combined");
    // Number has no known additivity (Flint 1.1): rows average.
    const averaged = assembleNtcharts(chart("Bar Chart", rows, { x: { field: "c" }, y: { field: "v" } }, { c: "Category", v: "Number" }));
    expect(averaged.data.series[0].values!.map((p) => p.y)).toEqual([3, 5]);
  });

  it("merge a facet into each bar with a warning rather than keep its last row", () => {
    const rows = ["East", "West"].flatMap((r) => [{ s: "A", r, v: 10 }, { s: "B", r, v: 20 }]);
    const out = assembleNtcharts(chart("Bar Chart", rows, { x: { field: "s" }, y: { field: "v" }, column: { field: "r" } }, { s: "Category", r: "Category", v: "Amount" }));
    expect(out.data.series[0].values!.map((p) => p.y)).toEqual([20, 40]);
    expect(codes(out)).toContain("facets-merged");
  });

  it("lay many dates out as bands, trimmed to what the width draws", () => {
    const rows = Array.from({ length: 100 }, (_, i) => ({ d: new Date(Date.UTC(2020, 0, 1 + i * 3)).toISOString().slice(0, 10), v: i + 1 }));
    const out = assembleNtcharts(chart("Bar Chart", rows, { x: { field: "d" }, y: { field: "v" } }, { d: "Date", v: "Quantity" }));
    const labels = out.x_axis!.labels!;
    expect(labels.length).toBeLessThanOrEqual(T.width / 2);
    expect(labels.length).toBe(out.data.series[0].values!.length);
    expect(codes(out)).toContain("overflow");
  });

  it("put dates in time order and name months as months", () => {
    const rows = [{ m: "2024-03-01", v: 3 }, { m: "2024-01-01", v: 1 }, { m: "2024-02-01", v: 2 }];
    const out = assembleNtcharts(chart("Bar Chart", rows, { x: { field: "m" }, y: { field: "v" } }, { m: "Date", v: "Quantity" }));
    expect(out.x_axis!.labels).toEqual(["2024-01", "2024-02", "2024-03"]);
    expect(out.data.series[0].values!.map((p) => p.y)).toEqual([1, 2, 3]);
  });

  it("decline two categorical axes, naming them", () => {
    const message = errorOf(chart("Bar Chart", [{ a: "x", b: "y" }], { x: { field: "a" }, y: { field: "b" } }, { a: "Category", b: "Category" }));
    expect(message).toContain("needs one quantitative axis");
    expect(message).toContain("a (nominal)");
  });
});

describe("scatter plots", () => {
  it("draw a categorical y beside a measured x transposed, and say so", () => {
    const rows = ["p", "q", "r"].map((g, i) => ({ g, v: i * 10 }));
    const out = assembleNtcharts(chart("Scatter Plot", rows, { x: { field: "v" }, y: { field: "g" } }, { g: "Category", v: "Quantity" }));
    expect(out.x_axis?.type).toBe("category");
    expect(out.x_axis?.labels).toEqual(["p", "q", "r"]);
    expect(out.data.series[0].values!.map((p) => p.y)).toEqual([0, 10, 20]);
    expect(codes(out)).toContain("axes-transposed");
  });

  it("decline two categorical axes", () => {
    expect(errorOf(chart("Scatter Plot", [{ a: "x", b: "y" }], { x: { field: "a" }, y: { field: "b" } }, { a: "Category", b: "Category" })))
      .toContain("needs a measured axis");
  });

  it("draw dates along x on a time axis rather than as NaN", () => {
    const rows = [{ d: "2024-03-01", v: 1 }, { d: "2024-01-01", v: 2 }];
    const out = assembleNtcharts(chart("Scatter Plot", rows, { x: { field: "d" }, y: { field: "v" } }, { d: "Date", v: "Quantity" }));
    expect(out.x_axis?.format?.kind).toBe("time");
    expect(out.data.series[0].values!.map((p) => p.x)).toEqual([Date.UTC(2024, 2, 1), Date.UTC(2024, 0, 1)]);
  });
});

describe("numeric X axes", () => {
  const rows = (a: number, b: number) => [{ x: a, y: 1 }, { x: b, y: 2 }];
  const enc = { x: { field: "x" }, y: { field: "y" } };
  const st = { x: "Quantity", y: "Quantity" };
  it("say when a line chart's X axis has to reach 0 far from the data", () => {
    expect(codes(assembleNtcharts(chart("Line Chart", rows(1000, 1100), enc, st)))).toContain("x-axis-from-zero");
    expect(codes(assembleNtcharts(chart("Line Chart", rows(-5, 10), enc, st)))).not.toContain("x-axis-from-zero");
  });

  it("say nothing for scatter plots, whose X axis fits the points", () => {
    expect(codes(assembleNtcharts(chart("Scatter Plot", rows(1000, 1100), enc, st)))).not.toContain("x-axis-from-zero");
  });
});

describe("lines, candlesticks and facets", () => {
  it("decline a line chart whose y is not a measure", () => {
    expect(errorOf(chart("Line Chart", [{ v: 1, s: "a" }, { v: 2, s: "b" }], { x: { field: "v" }, y: { field: "s" } }, { v: "Quantity", s: "Category" })))
      .toContain("measures along y");
  });

  it("decline a faceted candlestick rather than merge its instruments", () => {
    const rows = ["AAA", "BBB"].map((t) => ({ t, d: "2024-01-02", o: 1, h: 2, l: 0.5, c: 1.5 }));
    expect(errorOf(chart("Candlestick Chart", rows,
      { x: { field: "d" }, open: { field: "o" }, high: { field: "h" }, low: { field: "l" }, close: { field: "c" }, column: { field: "t" } },
      { d: "Date", o: "Price", h: "Price", l: "Price", c: "Price", t: "Category" }))).toContain("draws one instrument");
  });

  it("colour flattened facets apart", () => {
    const rows = ["N", "S"].flatMap((r) => ["Jan", "Feb"].map((m, i) => ({ r, m, v: i })));
    const out = assembleNtcharts(chart("Line Chart", rows, { x: { field: "m" }, y: { field: "v" }, column: { field: "r" } }, { r: "Category", m: "Month", v: "Quantity" }));
    const colors = out.data.series.map((s) => s.color);
    expect(colors.every(Boolean)).toBe(true);
    expect(new Set(colors).size).toBe(out.data.series.length);
  });
});

describe("sparklines and histograms", () => {
  it("spread a sparkline over its width, interpolating or averaging", () => {
    expect(resample([0, 10], 5)).toEqual([0, 2.5, 5, 7.5, 10]);
    expect(resample([1, 2, 3, 4, 5, 6], 3)).toEqual([1.5, 3.5, 5.5]);
  });

  it("draw several sparklines as lines on one chart, not one series run together", () => {
    const rows = ["a", "b"].flatMap((k) => [1, 2, 5].map((i) => ({ k, d: `2024-01-0${i}`, v: i })));
    const out = assembleNtcharts(chart("Sparkline", rows, { x: { field: "d" }, y: { field: "v" }, color: { field: "k" } }, { k: "Category", d: "Date", v: "Quantity" }));
    expect(out.type).toBe("timeseries");
    expect(out.data.series).toHaveLength(2);
    expect(codes(out)).toContain("chart-type-approximated");
  });

  it("name histogram bins by their starts, compactly, in ASCII", () => {
    expect(edgeLabels([20500, 38500, 999600])).toEqual(["20.5k", "38.5k", "1M"]);
    // Narrow bars take fewer figures, not a cut name ("18." from 18.6).
    expect(edgeLabels([12, 18.6, 25.3], 3)).toEqual(["12", "19", "25"]);
    const rows = Array.from({ length: 50 }, (_, i) => ({ v: 12 + i * 1.3 }));
    const out = assembleNtcharts(chart("Histogram", rows, { x: { field: "v" } }, { v: "Quantity" }));
    for (const label of out.x_axis!.labels!) expect(label).toMatch(/^[-0-9.kMB]+$/);
  });
});

describe("date category names", () => {
  it("are as short as the values allow", () => {
    expect(categoryLabels(["2020-01-01", "2021-01-01"])).toEqual(["2020", "2021"]);
    expect(categoryLabels(["2020-01-01", "2020-02-01"])).toEqual(["2020-01", "2020-02"]);
    expect(categoryLabels(["2020-01-01", "2020-01-15"])).toEqual(["2020-01-01", "2020-01-15"]);
    expect(categoryLabels(["North", "South"])).toEqual(["North", "South"]);
  });
});

describe("axis names in the fewest characters that fit", () => {
  const iso = (y: number, m: number, d = 1, h = 0) => new Date(Date.UTC(y, m, d, h)).toISOString().slice(0, h ? 16 : 10);
  const plan = (keys: string[], slots: ReturnType<typeof barSlots>, display = keys) =>
    planTicks(keys, display, tickKind(keys, true), slots).ticks.filter(Boolean);

  it("name months by the calendar, the year where it starts, as many as fit", () => {
    const months = Array.from({ length: 24 }, (_, i) => iso(2020, i));
    expect(plan(months, barSlots(24, 64))).toEqual(["2020", "Apr", "Jul", "Oct", "2021", "Apr", "Jul", "Oct"]);
    expect(plan(months.slice(0, 7), centerSlots(7, 55))).toEqual(["2020", "Feb", "Mar", "Apr", "May", "Jun", "Jul"]);
  });

  it("name days, weeks and hours by their own periods", () => {
    const days = Array.from({ length: 31 }, (_, i) => iso(2024, 2, 1 + i));
    expect(plan(days, barSlots(31, 64))).toEqual(["Mar", "8", "15", "22"]);
    const hours = Array.from({ length: 24 }, (_, i) => `2024-03-14T${String(i).padStart(2, "0")}:00`);
    expect(plan(hours, barSlots(24, 64))[0]).toBe("3/14");
    expect(plan(hours, barSlots(24, 64))).toContain("06");
  });

  it("name a year where its first date falls, even on an irregular axis", () => {
    const dates = Array.from({ length: 24 }, (_, i) => iso(2020, 0, 15 + 45 * i));
    expect(plan(dates, barSlots(24, 64))).toEqual(["Jan", "Jul", "2021", "Jul", "2022", "Jul"]);
  });

  it("name hours of the day every few hours, however the column is typed", () => {
    const clock = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, "0")}:00`);
    expect(tickKind(clock, false)).toBe("ordinal");
    expect(plan(clock, barSlots(24, 64)).slice(0, 4)).toEqual(["00", "02", "04", "06"]);
  });

  it("name numbers at nice values and years every five", () => {
    const ranks = Array.from({ length: 30 }, (_, i) => String(i + 1));
    expect(plan(ranks, barSlots(30, 64))).toEqual(["2", "4", "6", "8", "10", "12", "14", "16", "18", "20", "22", "24", "26", "28", "30"]);
    const years = Array.from({ length: 30 }, (_, i) => String(1995 + i));
    expect(plan(years, barSlots(30, 64))).toEqual(["1995", "2000", "2005", "2010", "2015", "2020"]);
  });

  it("name every k-th ordered category, and cut nominal ones with an ellipsis rather than skip them", () => {
    const stages = Array.from({ length: 30 }, (_, i) => `Stage ${i + 1}`);
    expect(plan(stages, centerSlots(30, 64))[1]).toBe("Stage 8");
    const names = ["Electronics", "Clothing", "Toys"];
    const nominal = planTicks(names, names, tickKind(names, false), barSlots(3, 18));
    expect(nominal).toEqual({ ticks: ["Elec…", "Clot…", "Toys"], shortened: true });
  });

  it("keep a date axis sorted by value from being thinned as if it ran in time", () => {
    expect(tickKind(["2020-03-01", "2020-01-01"], true)).toBe("ordinal");
  });

  it("turn bars whose names do not fit, and print bar values where they fit", () => {
    const rows = ["Electronics", "Home & Garden", "Automotive", "Toys"].map((c, i) => ({ c, v: 10 * (i + 1) }));
    const out = assembleNtcharts(chart("Bar Chart", rows, { x: { field: "c" }, y: { field: "v" } }, { c: "Category", v: "Quantity" }, { baseSize: { width: 30, height: 12 } }));
    expect(out.options?.orientation).toBe("horizontal");
    expect(codes(out)).toContain("bars-horizontal");
    expect(out.options?.show_values).toBe(true);
    const kept = assembleNtcharts(chart("Bar Chart", rows, { x: { field: "c" }, y: { field: "v" } }, { c: "Category", v: "Quantity" },
      { baseSize: { width: 30, height: 12 }, chartProperties: { orientation: "vertical", valueLabels: false } }));
    expect(kept.options?.orientation).toBeUndefined();
    expect(kept.options?.show_values).toBeUndefined();
    expect(kept.x_axis!.tick_labels!.every((t: string) => t.length <= 6)).toBe(true);
  });
});

describe("chart types added for flint 1.1", () => {
  const sales = ["North", "South"].flatMap((r) => ["Online", "Store", "Phone"].map((c, i) => ({ r, c, v: 10 * (i + 1) + (r === "South" ? 5 : 0) })));

  it("set grouped bars side by side: one coloured bar per group, an empty bar between categories", () => {
    const out = assembleNtcharts(chart("Grouped Bar Chart", sales, { x: { field: "r" }, y: { field: "v" }, group: { field: "c" } }, { r: "Category", c: "Category", v: "Quantity" }));
    expect(out.x_axis!.labels).toEqual(["North", "", "", "", "South", "", ""]);
    expect(out.data.series.map((s) => s.name)).toEqual(["Online", "Store", "Phone"]);
    expect(out.data.series[1].values!.map((p) => p.y)).toEqual([0, 20, 0, 0, 0, 25, 0]);
    expect(new Set(out.data.series.map((s) => s.color)).size).toBe(3);
  });

  it("keep a lane for every group under a global dodge, and only a category's own under a local one", () => {
    const sparse = sales.filter((r) => !(r.r === "North" && r.c === "Store"));
    const draw = (dodge: string) => assembleNtcharts(chart("Grouped Bar Chart", sparse, { x: { field: "r" }, y: { field: "v" }, group: { field: "c" } },
      { r: "Category", c: "Category", v: "Quantity" }, { chartProperties: { dodge } }));
    expect(draw("global").x_axis!.labels).toHaveLength(7);
    expect(draw("local").x_axis!.labels).toHaveLength(6);
  });

  it("combine a grouping with too many values to tell apart, and say so", () => {
    const rows = Array.from({ length: 40 }, (_, i) => ({ c: ["a", "b"][i % 2], g: i, v: 1 }));
    const out = assembleNtcharts(chart("Grouped Bar Chart", rows, { x: { field: "c" }, y: { field: "v" }, group: { field: "g" } }, { c: "Category", g: "Quantity", v: "Quantity" }));
    expect(codes(out)).toContain("too-many-groups");
    expect(out.data.series[0].values!.map((p) => p.y)).toEqual([20, 20]);
  });

  it("set a pyramid's two groups back to back on a symmetric axis, first category at the bottom", () => {
    const rows = ["0-9", "10-19"].flatMap((a, i) => [{ a, s: "M", n: 10 + i }, { a, s: "F", n: 20 + i }]);
    const out = assembleNtcharts(chart("Pyramid Chart", rows, { y: { field: "a" }, x: { field: "n" }, color: { field: "s" } }, { a: "Category", s: "Category", n: "Quantity" }));
    expect(out.options?.orientation).toBe("horizontal");
    expect(out.x_axis!.labels).toEqual(["10-19", "0-9"]);
    expect(out.data.series[0].values!.map((p) => p.y)).toEqual([-11, -10]);
    expect([out.y_axis!.min, out.y_axis!.max]).toEqual([-21, 21]);
    expect((out._warnings ?? []).find((w: any) => w.code === "pyramid-sides")?.message).toBe("M runs left, F right");
  });

  it("estimate densities as flint does and label their small values", () => {
    const curve = kde([0, 0, 0, 1, -1], 200, 1, { min: -10, max: 10 });
    const area = curve.reduce((s, p, i) => (i ? s + ((p.y + curve[i - 1].y) / 2) * (p.x - curve[i - 1].x) : s), 0);
    expect(area).toBeCloseTo(1, 2);
    const out = assembleNtcharts(chart("Density Plot", Array.from({ length: 50 }, (_, i) => ({ v: 100 + i })), { x: { field: "v" } }, { v: "Quantity" }));
    expect(out.y_axis?.format).toEqual({ kind: "number", precision: 3 });
  });

  it("draw a range area as its two edges in one colour, the axis fitted to both", () => {
    const rows = [1, 2, 3].map((x) => ({ x, lo: 10 + x, hi: 20 + x }));
    const out = assembleNtcharts(chart("Range Area Chart", rows, { x: { field: "x" }, y: { field: "lo" }, y2: { field: "hi" } }, { x: "Quantity", lo: "Quantity", hi: "Quantity" }));
    expect(out.data.series.map((s) => s.values!.map((p) => p.y))).toEqual([[11, 12, 13], [21, 22, 23]]);
    expect(out.data.series[0].color).toBe(out.data.series[1].color);
    expect(out.y_axis!.min!).toBeLessThan(11);
    expect(out.y_axis!.max!).toBeGreaterThan(23);
  });

  it("fit regressions by flint's methods and draw the fit beneath the points", () => {
    const line = fitCurve([[0, 1], [1, 3], [2, 5]], "linear", 3)!;
    expect(line.fit(10)).toBeCloseTo(21);
    const exp = fitCurve([0, 1, 2, 3].map((x) => [x, 2 * Math.exp(0.5 * x)]), "exp", 3)!;
    expect(exp.fit(4)).toBeCloseTo(2 * Math.exp(2));
    const rows = Array.from({ length: 20 }, (_, i) => ({ x: i, y: 3 * i + (i % 3) }));
    const out = assembleNtcharts(chart("Regression", rows, { x: { field: "x" }, y: { field: "y" } }, { x: "Quantity", y: "Quantity" }));
    expect(out.data.series[0]).toMatchObject({ name: "trend", color: "#ee6666", symbol: "·" });
    expect(out.data.series[0].values!.length).toBe(out.width);
    expect(out.data.series[1].values).toHaveLength(20);
  });
});

describe("dates on a line chart's X axis", () => {
  const months = ["2026-01-01", "2026-02-01", "2026-03-01", "2026-04-01"].map((d, i) => ({ d, v: i + 1 }));
  const draw = (rows: Record<string, unknown>[]) =>
    assembleNtcharts(chart("Line Chart", rows, { x: { field: "d" }, y: { field: "v" } }, { d: "Date", v: "Quantity" }));

  it("put a regular series on date slots, named by the calendar under each point", () => {
    const out = draw(months);
    expect(out.x_axis).toMatchObject({ type: "category", labels: months.map((m) => m.d), tick_labels: ["2026", "Feb", "Mar", "Apr"] });
  });

  it("keep a time axis for irregular dates, gaps or more points than labels fit", () => {
    expect(draw([...months.slice(0, 3), { d: "2026-03-20", v: 9 }]).type).toBe("timeseries");
    expect(draw(months.filter((_, i) => i !== 1)).type).toBe("timeseries");
    const daily = Array.from({ length: 90 }, (_, i) => ({ d: new Date(Date.UTC(2026, 0, 1 + i)).toISOString().slice(0, 10), v: i }));
    expect(draw(daily).type).toBe("timeseries");
  });

  it("recognise yearly, monthly and fixed-step cadences", () => {
    expect(regularDates(["2020-01-01", "2021-01-01", "2022-01-01"])).toHaveLength(3);
    expect(regularDates(["2020-01-01", "2020-04-01", "2020-07-01"])).toHaveLength(3);
    expect(regularDates(["2020-01-01", "2020-01-08", "2020-01-15"])).toHaveLength(3);
    expect(regularDates(["2020-01-01", "2020-01-08", "2020-01-16"])).toBeUndefined();
  });
});

describe("styling a terminal chart", () => {
  const rows = ["a", "b"].flatMap((g) => [1, 2, 3].map((x) => ({ g, x, y: x * (g === "a" ? 1 : -1) })));
  const enc = { x: { field: "x" }, y: { field: "y" } };
  const st = { x: "Quantity", y: "Quantity", g: "Category" };

  it("take a Flint theme's series inks", () => {
    const one = assembleNtcharts({ ...chart("Bar Chart", [{ c: "a", v: 1 }, { c: "b", v: 2 }], { x: { field: "c" }, y: { field: "v" } }, { c: "Category", v: "Quantity" }), theme_spec: "economist" } as any);
    expect(one.data.series[0].color).toBe("#006ba2");
    const two = assembleNtcharts({ ...chart("Scatter Plot", rows, { ...enc, color: { field: "g" } }, st), theme_spec: "economist" } as any);
    expect(two.data.series.map((s) => s.color)).toEqual(["#006ba2", "#3ebcd2"]);
  });

  it("set scatter series apart by symbol and draw zero where the range crosses it", () => {
    const out = assembleNtcharts(chart("Scatter Plot", rows, { ...enc, color: { field: "g" } }, st));
    expect(out.data.series.map((s) => s.symbol)).toEqual(["•", "×"]);
    expect(out.options?.zero_line).toBe(true);
    const positive = assembleNtcharts(chart("Line Chart", rows.filter((r) => r.y > 0), enc, st));
    expect(positive.options?.zero_line).toBeUndefined();
    expect([out.options?.nice_ticks, positive.options?.nice_ticks]).toEqual([true, true]);
  });

  it("pass the legend position through", () => {
    const out = assembleNtcharts(chart("Scatter Plot", rows, { ...enc, color: { field: "g" } }, st, { chartProperties: { legend: "bottom" } }));
    expect(out.options?.legend_position).toBe("bottom");
  });
});

describe("zero baselines", () => {
  it("keep negative points on the plot when the axis includes zero", () => {
    const rows = [{ x: 1, y: -3 }, { x: 2, y: 4 }];
    for (const type of ["Scatter Plot", "Line Chart"]) {
      const out = assembleNtcharts(chart(type, rows, { x: { field: "x" }, y: { field: "y" } }, { x: "Quantity", y: "Quantity" }));
      expect(out.y_axis?.min, type).toBeUndefined();
    }
    const below = assembleNtcharts(chart("Scatter Plot", [{ x: 1, y: -3 }, { x: 2, y: -1 }], { x: { field: "x" }, y: { field: "y" } }, { x: "Quantity", y: "Quantity" }));
    expect(below.y_axis?.max).toBe(0);
  });

  it("never pad a fitted range across zero", () => {
    const rows = [0.8, 3, 7.5, 12, 18.6].map((y, x) => ({ x, y }));
    const out = assembleNtcharts(chart("Scatter Plot", rows, { x: { field: "x" }, y: { field: "y" } }, { x: "Quantity", y: "Quantity" }));
    expect(out.y_axis?.min).toBe(0);
    expect(out.y_axis?.max).toBeGreaterThan(18.6);
  });
});
