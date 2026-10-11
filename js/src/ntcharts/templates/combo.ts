import { resolveChannelSemantics, computePaddedDomain } from "flint-chart/core";
import type { ChartTemplateDef } from "flint-chart/core";
import type { NtInstantiateContext } from "../assemble.js";
import { formatSpecToNt } from "../format.js";

// Matches Flint's Chart.js Combo Chart contract. No secondaryY extension:
// x names categories, y supplies bars, lineField selects the right-hand line.
export const ntComboChartDef: ChartTemplateDef = {
  chart: "Combo Chart",
  template: { mark: "bar" },
  channels: ["x", "y", "column", "row"],
  markCognitiveChannel: "length",
  declareLayoutMode: () => ({ axisFlags: { x: { banded: true } }, paramOverrides: { minStep: 2 } }),
  instantiate: (_spec: any, rawCtx: any) => {
    const ctx = rawCtx as NtInstantiateContext;
    const { emit, channelSemantics: cs, table, chartProperties: props, encodings } = ctx;
    const x = cs.x?.field, y = cs.y?.field;
    if (!x || !y || !table.length) throw new Error("Combo Chart requires x, y, and nonempty data");
    if (encodings.column || encodings.row) throw new Error("terminal Combo Chart does not support facets");
    if (Object.values(encodings).some(e => e?.aggregate)) {
      throw new Error("terminal Combo Chart sums duplicate categories; pre-aggregate data for other reductions");
    }
    // Flint's fallback examines fields of the first row and uses a 90%
    // numeric threshold; explicit lineField wins when that field exists.
    const numeric = (field: string) => {
      const values = table.map(r => r[field]).filter(v => v != null && v !== "");
      return values.length > 0 && values.filter(v => Number.isFinite(Number(v))).length / values.length >= .9;
    };
    const lineField = typeof props?.lineField === "string" && props.lineField in table[0]
      ? props.lineField
      : Object.keys(table[0]).find(k => k !== x && k !== y && numeric(k));
    if (lineField === y) throw new Error("Combo Chart lineField must differ from the bar field");
    const labels = [...new Set(table.filter(r => r[x] != null).map(r => String(r[x])))];
    const order = cs.x?.ordinalSortOrder;
    if (order?.length) {
      const ranks = new Map(order.map((v, i) => [String(v), i]));
      labels.sort((a, b) => (ranks.get(a) ?? Infinity) - (ranks.get(b) ?? Infinity));
    }
    if (!labels.length) throw new Error("Combo Chart requires at least one category");
    const points = (field: string) => {
      const sums = new Map<string, number>();
      let invalid = 0;
      for (const row of table) {
        if (row[x] == null || row[field] == null) continue;
        const value = Number(row[field]);
        if (!Number.isFinite(value)) { invalid++; continue; }
        const key = String(row[x]);
        const sum = (sums.get(key) ?? 0) + value;
        if (!Number.isFinite(sum)) throw new Error(`Combo Chart sum overflows for ${field}`);
        sums.set(key, sum);
      }
      if (invalid) ctx.warn({ severity: "warning", code: "invalid-combo-value",
        message: `omitted ${invalid} nonnumeric value(s) from ${field}` });
      return labels.filter(label => sums.has(label)).map(label => ({ x: label, y: sums.get(label)! }));
    };
    emit.type = "bar";
    emit.x_axis = { type: "category", title: x, labels };
    emit.y_axis = { title: y, ...(ctx.ntFormatY ? { format: ctx.ntFormatY } : {}) };
    // Leave the bar range unpinned: ntcharts includes zero and negative bars.
    emit.data.series = [{ name: y, type: "bar", values: points(y) }];
    if (lineField) {
      const values = points(lineField);
      const lineCS = resolveChannelSemantics({ y: { field: lineField, type: "quantitative" } }, table,
        ctx.semanticTypes ?? {}, table).y;
      const format = formatSpecToNt(lineCS?.format, ctx.warn);
      emit.y2_axis = { title: lineField, ...(format ? { format } : {}) };
      const range = computePaddedDomain(values.map(p => p.y), .05);
      if (range) [emit.y2_axis.min, emit.y2_axis.max] = range;
      emit.data.series.push({ name: lineField, type: "line", y_axis: "right", values });
      emit.options = { show_legend: true };
    } else {
      // Plain barchart consumes positional slots, unlike ComboModel's sparse
      // category points. Keep missing categories in place for the fallback.
      const bars = new Map(emit.data.series[0].values!.map(p => [p.x, p.y]));
      emit.data.series[0].values = labels.map(x => ({ x, y: bars.get(x) ?? 0 }));
    }
    if (props?.cornerRadius) ctx.warn({ severity: "info", code: "chart-property-unsupported",
      message: "terminal Combo Chart cannot draw rounded bar corners" });
  },
};
