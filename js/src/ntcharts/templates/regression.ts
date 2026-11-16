import type { ChartTemplateDef } from "flint-chart/core";
import { makeCartesianPivot } from "flint-chart/core";
import { ntScatterPlotDef } from "./scatter.js";
import { rowGroups } from "../groups.js";
import type { NtInstantiateContext } from "../assemble.js";
import type { NtSeries } from "../types.js";

type Fit = (x: number) => number;

function linear(points: number[][]): Fit {
  const n = points.length;
  let sx = 0, sy = 0, sxy = 0, sxx = 0;
  for (const [x, y] of points) { sx += x; sy += y; sxy += x * y; sxx += x * x; }
  const slope = (n * sxy - sx * sy) / (n * sxx - sx * sx) || 0;
  const intercept = (sy - slope * sx) / n;
  return (x) => slope * x + intercept;
}

function polynomial(points: number[][], order: number): Fit {
  const k = order + 1;
  const a = Array.from({ length: k }, () => new Array(k + 1).fill(0));
  for (const [x, y] of points) {
    const p = [1];
    for (let i = 1; i <= 2 * order; i++) p.push(p[i - 1] * x);
    for (let i = 0; i < k; i++) {
      a[i][k] += y * p[i];
      for (let j = 0; j < k; j++) a[i][j] += p[i + j];
    }
  }
  for (let c = 0; c < k; c++) {
    let m = c;
    for (let r = c + 1; r < k; r++) if (Math.abs(a[r][c]) > Math.abs(a[m][c])) m = r;
    [a[c], a[m]] = [a[m], a[c]];
    const pivot = a[c][c];
    if (Math.abs(pivot) < 1e-12) continue;
    for (let j = c; j <= k; j++) a[c][j] /= pivot;
    for (let r = 0; r < k; r++) {
      if (r === c) continue;
      const f = a[r][c];
      for (let j = c; j <= k; j++) a[r][j] -= f * a[c][j];
    }
  }
  const coeffs = a.map((row) => row[k]);
  return (x) => coeffs.reduce((sum, c, i) => sum + c * x ** i, 0);
}

/**
 * The fitted curve, its domain and the points it was fitted to, by flint's
 * methods: linear, log (y = a + b·ln x), exp (ln y linear in x), pow
 * (ln y linear in ln x), quad, poly.
 */
export function fitCurve(points: number[][], method: string, order: number): { fit: Fit; from: number; to: number } | undefined {
  const usable = points.filter(([x, y]) =>
    Number.isFinite(x) && Number.isFinite(y) && (method !== "log" || x > 0) && (method !== "exp" || y > 0) && (method !== "pow" || (x > 0 && y > 0)));
  if (usable.length < 2) return undefined;
  const xs = usable.map(([x]) => x);
  const domain = { from: Math.min(...xs), to: Math.max(...xs) };
  switch (method) {
    case "log": { const f = linear(usable.map(([x, y]) => [Math.log(x), y])); return { fit: (x) => f(Math.log(x)), ...domain }; }
    case "exp": { const f = linear(usable.map(([x, y]) => [x, Math.log(y)])); return { fit: (x) => Math.exp(f(x)), ...domain }; }
    case "pow": { const f = linear(usable.map(([x, y]) => [Math.log(x), Math.log(y)])); return { fit: (x) => Math.exp(f(Math.log(x))), ...domain }; }
    case "quad": return { fit: polynomial(usable, 2), ...domain };
    case "poly": return { fit: polynomial(usable, order), ...domain };
    default: return { fit: linear(usable), ...domain };
  }
}

// ntcharts draws a scatter plot's points but cannot lay a line over them, so
// each fitted curve is drawn as a run of points, one per column, beneath the
// data. Without a colour the curve is red, as flint draws it; with one, each
// group's curve takes the group's colour.
export const ntRegressionDef: ChartTemplateDef = {
  ...ntScatterPlotDef,
  chart: "Regression",
  channels: ["x", "y", "size", "color", "column", "row"],
  instantiate: (spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { channelSemantics: cs } = ctx;
    if (cs.x?.type !== "quantitative" || cs.y?.type !== "quantitative") {
      throw new Error(`A Regression fits y to x and needs both to be numbers; got x = ${cs.x?.field} (${cs.x?.type}), y = ${cs.y?.field} (${cs.y?.type}).`);
    }
    ntScatterPlotDef.instantiate(spec, rawCtx);
    const { emit } = ctx;
    const method = String(ctx.chartProperties?.regressionMethod ?? "linear");
    const order = Number(ctx.chartProperties?.polyOrder ?? 3);
    const [lo, hi] = [emit.y_axis?.min, emit.y_axis?.max];
    const groups = rowGroups(ctx);
    const curves: NtSeries[] = [];
    const points: NtSeries[] = [];
    for (const g of groups) {
      const xy = g.rows.map((r) => [Number(r[cs.x!.field]), Number(r[cs.y!.field])]);
      points.push({ name: g.name || cs.y!.field, values: xy.filter(([x, y]) => Number.isFinite(x) && Number.isFinite(y)).map(([x, y]) => ({ x, y })), ...(g.color ? { color: g.color } : {}) });
      const curve = fitCurve(xy, method, order);
      if (!curve) continue;
      const n = Math.max(2, emit.width);
      const values = Array.from({ length: n }, (_, i) => {
        const x = curve.from + ((curve.to - curve.from) * i) / (n - 1);
        return { x, y: curve.fit(x) };
      }).filter((p) => Number.isFinite(p.y) && (lo == null || p.y >= lo) && (hi == null || p.y <= hi));
      // Named as its group, so a legend names each group once; ungrouped, "trend".
      curves.push({ name: g.name || "trend", values, color: g.color ?? "#ee6666", symbol: "·" });
    }
    // Curves first, so the data points are drawn over them.
    emit.data.series = [...curves, ...points];
    emit.options = { ...emit.options, show_legend: groups.length > 1 };
    ctx.warn({ severity: "info", code: "chart-type-approximated",
      message: `the ${method} fit is drawn as a row of points: the terminal scatter cannot draw a line over its points` });
  },
  pivot: makeCartesianPivot({ transpose: [["x", "y"]], permute: [["x", "y", "color"]], shift: ["color"] }),
};
