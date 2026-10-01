import {
  resolveChannelSemantics, computeZeroDecision, convertTemporalData,
  computeChannelBudgets, filterOverflow, computeLayout,
  applyPivot, applyEncodingOverrides, normalizeStaticSeries,
} from "flint-chart/core";
import type {
  ChartAssemblyInput, ChartTemplateDef, ChannelSemantics, ChartWarning,
  LayoutResult, AssembleOptions, LayoutDeclaration,
} from "flint-chart/core";
import { ntGetTemplateDef, ntSupportedChartTypes } from "./templates/index.js";
import { resolveBaseSizeShim, deriveStretchCapsShim, applyAggregationShim } from "./shims.js";
import { formatSpecToNt, d3TimeToGoLayout } from "./format.js";
import type { NtSpec, NtSpecOut, NtFormat } from "./types.js";

// maxStretch: 1 — terminal charts must not exceed the requested base size
// (terminal cells are a fixed budget, not a freely growable canvas like
// pixels); pass a `canvasSize` larger than `baseSize` in the chart spec to
// explicitly allow growth beyond the base.
//
// minSubplotSize: flint floors the plot size it budgets for at this value
// (default 60, a pixel number). Left at 60, a 20-cell-wide chart would be
// budgeted as if it were 60 wide and keep 60 categories. 4 is the smallest
// plot this backend emits (see the Math.max floors on emit.width/height).
const TERMINAL_OPTIONS: AssembleOptions = {
  minStep: 1, defaultBandSize: 3, stepPadding: 0.2, maxStretch: 1, minSubplotSize: 4,
};

// Fitted-domain padding used when `includeZero_y: false` overrides a decision
// that carried none (flint pads only decisions that already excluded zero).
// 0.05 is the fraction flint's own non-zero decisions use.
const OVERRIDE_DOMAIN_PAD = 0.05;

export interface NtInstantiateContext {
  channelSemantics: Record<string, ChannelSemantics>;
  layout: LayoutResult;
  table: Record<string, unknown>[];
  encodings: Record<string, any>;
  chartProperties?: Record<string, unknown>;
  canvasSize: { width: number; height: number };
  chartType: string;
  emit: NtSpec;               // template fills this in
  warn: (w: ChartWarning) => void;
  ntFormatX?: NtFormat;       // pre-translated axis formats
  ntFormatY?: NtFormat;
}

export function assembleNtcharts(input: ChartAssemblyInput): NtSpecOut {
  const warnings: ChartWarning[] = [];
  const warn = (w: ChartWarning) => warnings.push(w);
  const chartType = input.chart_spec.chartType;
  let template: ChartTemplateDef | undefined = ntGetTemplateDef(chartType);
  if (!template) {
    throw new Error(`Unknown chart type "${chartType}". Supported: ${ntSupportedChartTypes().join(", ")}`);
  }
  const semanticTypes = input.semantic_types ?? {};
  const rawData: Record<string, unknown>[] = (input.data as any).values ?? [];

  // Sizes: interpreted in CELLS throughout.
  const ceiling = input.chart_spec.canvasSize;
  const baseSize = resolveBaseSizeShim(input.chart_spec.baseSize, ceiling);
  const caps = deriveStretchCapsShim(baseSize, ceiling, TERMINAL_OPTIONS.maxStretch);

  // PRE-PHASE
  const normalized = normalizeStaticSeries(input.chart_spec.encodings ?? {}, rawData, semanticTypes);
  let data = normalized.data;

  // Preliminary typing so pivot/overrides see resolved types.
  const prelimConverted = convertTemporalData(data, semanticTypes);
  const prelim = resolveChannelSemantics(normalized.encodings, data, semanticTypes, prelimConverted);
  const typedEncodings: Record<string, any> = {};
  for (const [ch, enc] of Object.entries(normalized.encodings)) {
    if (!enc) continue;
    typedEncodings[ch] = { ...(enc as object), type: (enc as any).type ?? prelim[ch]?.type };
  }

  // Pivot (may re-dispatch chart type within our registry).
  const pivoted = applyPivot(template, typedEncodings, data, input.chart_spec.chartProperties, ntGetTemplateDef);
  if (pivoted.chartType && pivoted.chartType !== template.chart) {
    const next = ntGetTemplateDef(pivoted.chartType);
    if (next) template = next;
  }
  const composed = applyEncodingOverrides(template, pivoted.encodings, input.chart_spec.chartProperties);
  const encodings = template.normalizeEncodings?.(composed, data) ?? composed;
  data = applyAggregationShim(encodings, data);

  // PHASE 0: semantics + zero.
  const convertedData = convertTemporalData(data, semanticTypes);
  const channelSemantics = resolveChannelSemantics(encodings, data, semanticTypes, convertedData);
  const markType = (template.template as any)?.mark ?? "point";
  for (const axis of ["x", "y"] as const) {
    const cs = channelSemantics[axis];
    if (cs?.type === "quantitative") {
      const nums = convertedData.map((r: any) => Number(r[cs.field])).filter((n: number) => !Number.isNaN(n));
      cs.zero = computeZeroDecision(cs.semanticAnnotation.semanticType, axis, markType, nums);
    }
  }
  applyAxisProperties(template, channelSemantics, input.chart_spec.chartProperties, warn);

  // STEP 0a/0c + PHASE 1: layout.
  const declaration: LayoutDeclaration =
    template.declareLayoutMode?.(channelSemantics, data, input.chart_spec.chartProperties) ?? {};
  const options: AssembleOptions = { ...TERMINAL_OPTIONS, ...declaration.paramOverrides, ...caps };
  const budgets = computeChannelBudgets(channelSemantics, declaration, convertedData, baseSize, options);
  // NOTE (signature delta): filterOverflow's last parameter is a Set<string>
  // of mark types (per node_modules/flint-chart/dist/core/index.d.ts), not an
  // array as the task brief's transcription suggested.
  const overflow = filterOverflow(channelSemantics, declaration, encodings, convertedData, budgets, new Set([markType]));
  // `truncations` restates every entry of `warnings` with layout bookkeeping
  // (keptValues, placeholder) attached; surfacing both would report each
  // overflow twice.
  warnings.push(...overflow.warnings);
  // NOTE (signature delta): OverflowResult exposes the post-overflow rows as
  // `filteredData`, not `values` (the brief's field name was wrong).
  const layout = computeLayout(channelSemantics, declaration, overflow.filteredData, baseSize, options, budgets.facetGrid);

  // Axis formats (translated once; templates place them).
  const ntFormatX = axisFormat(channelSemantics.x, warn);
  const ntFormatY = axisFormat(channelSemantics.y, warn);

  // PHASE 2: instantiate into an NtSpec.
  //
  // The plot fills the cells it was asked for. For a chart whose X axis is
  // continuous (line, scatter, sparkline, a heatmap with numeric-looking
  // columns) flint's layout shrinks the plot — it runs a mark-density and
  // aspect-ratio-banking calculation designed for pixel canvases, and returns
  // about 45% of the requested width. A terminal window is the budget, so an
  // unused column is wasted; take the larger of flint's size and the request.
  // Banded charts already fill the request, and where a canvasSize ceiling
  // allows growth flint's larger size still wins. Without a ceiling the
  // stretch cap of 1 keeps flint's size at or below the request.
  const emit: NtSpec = {
    type: "bar", // template overwrites
    width: Math.max(8, Math.round(layout.subplotWidth), baseSize.width),
    height: Math.max(4, Math.round(layout.subplotHeight), baseSize.height),
    data: { series: [] },
  };
  if (input.chart_spec.title) emit.title = input.chart_spec.title;
  if (input.chart_spec.subtitle) emit.subtitle = input.chart_spec.subtitle;
  const ctx: NtInstantiateContext = {
    channelSemantics, layout, table: overflow.filteredData, encodings,
    chartProperties: input.chart_spec.chartProperties, canvasSize: baseSize,
    chartType: template.chart, emit, warn, ntFormatX, ntFormatY,
  };
  template.instantiate(emit as any, ctx as any);
  applyLogScale(emit, input.chart_spec.chartProperties, channelSemantics, warn);

  const result: NtSpecOut = { ...emit };
  if (warnings.length) result._warnings = warnings;
  result._width = emit.width;
  result._height = emit.height;
  return result;
}

// applyAxisProperties handles the per-axis chart properties upstream backends
// honour after the zero decision: `includeZero_<axis>` and `logScale_<axis>`.
//
// includeZero mirrors upstream (assemble: "markCognitiveChannel === position"
// charts only): an explicit true/false replaces the decision's `zero`. This
// backend can act on it for the Y axis of position charts, where templates
// pin y_axis.min/max; anywhere else it says so instead of ignoring the
// request — the X axis is never pinned, and bar charts keep a zero baseline
// because the terminal bar model always draws from zero.
//
// logScale is applied after the template has filled the spec; see applyLogScale.
function applyAxisProperties(
  template: ChartTemplateDef,
  channelSemantics: Record<string, ChannelSemantics>,
  chartProperties: Record<string, unknown> | undefined,
  warn: (w: ChartWarning) => void,
): void {
  if (!chartProperties) return;
  const positional = template.markCognitiveChannel === "position";
  for (const axis of ["x", "y"] as const) {
    const cs = channelSemantics[axis];
    const zeroChoice = chartProperties[`includeZero_${axis}`];
    if (zeroChoice === true || zeroChoice === false) {
      if (axis === "y" && positional && cs?.type === "quantitative" && cs.zero) {
        cs.zero = {
          ...cs.zero,
          zero: zeroChoice,
          domainPadFraction: zeroChoice ? cs.zero.domainPadFraction : cs.zero.domainPadFraction || OVERRIDE_DOMAIN_PAD,
        };
      } else {
        warn({
          severity: "info",
          code: "chart-property-unsupported",
          message: `includeZero_${axis} has no effect on a terminal ${template.chart}` +
            (axis === "x" ? ": the X axis range always follows the data." : ": its value axis always starts at zero."),
          channel: axis,
          field: cs?.field,
        } as ChartWarning);
      }
    }
  }
}

// applyLogScale turns `logScale_x` / `logScale_y` into ntcharts-spec axis
// scales. A log axis is only emitted where ntcharts can draw one and the data
// can sit on it; otherwise the chart stays linear and a warning says why,
// rather than silently ignoring the request.
//
//   - Y: line, scatter, timeseries and OHLC charts. X: line and scatter
//     (numeric X). Bars, heatmaps and sparklines have no log axis, and a time
//     axis cannot be logarithmic.
//   - Every value on the axis must be greater than zero. flint switches to a
//     symlog scale when the data contains zeros; ntcharts has no symlog.
//
// Upstream's zero-baseline and fitted-domain pins are linear-axis decisions
// (a log axis has no zero), so on a log Y axis they are dropped and ntcharts
// widens the range to whole decades around the data.
function applyLogScale(
  emit: NtSpec,
  chartProperties: Record<string, unknown> | undefined,
  channelSemantics: Record<string, ChannelSemantics>,
  warn: (w: ChartWarning) => void,
): void {
  if (!chartProperties) return;
  for (const axis of ["x", "y"] as const) {
    if (chartProperties[`logScale_${axis}`] !== true) continue;
    const reason = logScaleBlocker(emit, axis);
    if (reason) {
      warn({
        severity: "warning",
        code: "log-scale-unsupported",
        message: `logScale_${axis} was requested but ${reason}; drawn on a linear scale.`,
        channel: axis,
        field: channelSemantics[axis]?.field,
      } as ChartWarning);
      continue;
    }
    const target = axis === "x" ? (emit.x_axis ??= {}) : (emit.y_axis ??= {});
    target.scale = "log";
    if (axis === "y" && emit.y_axis) {
      delete emit.y_axis.min;
      delete emit.y_axis.max;
    }
  }
}

// logScaleBlocker returns why a log axis cannot be drawn on this spec, or
// undefined if it can.
function logScaleBlocker(emit: NtSpec, axis: "x" | "y"): string | undefined {
  const drawsLogY = ["line", "scatter", "timeseries", "ohlc"];
  const drawsLogX = ["line", "scatter"];
  if (!(axis === "y" ? drawsLogY : drawsLogX).includes(emit.type)) {
    if (axis === "x" && drawsLogY.includes(emit.type)) return "its X axis is time";
    return `a terminal ${emit.type} chart cannot draw a logarithmic axis`;
  }
  const values: number[] = [];
  for (const series of emit.data.series) {
    for (const p of series.values ?? []) values.push(axis === "y" ? p.y : (p.x as number));
    for (const p of series.ohlc ?? []) {
      if (axis === "y") values.push(p.o, p.h, p.l, p.c);
    }
  }
  if (values.some((v) => !(v > 0) || !Number.isFinite(v))) {
    return "the data has zero or negative values (flint would use a symlog scale, which terminal charts do not have)";
  }
  return undefined;
}

function axisFormat(cs: ChannelSemantics | undefined, warn: (w: ChartWarning) => void): NtFormat | undefined {
  if (!cs) return undefined;
  if (cs.type === "temporal" && cs.temporalFormat) {
    return { kind: "time", layout: d3TimeToGoLayout(cs.temporalFormat, warn) };
  }
  return formatSpecToNt(cs.format, warn);
}
