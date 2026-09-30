// ntcharts-spec v1 wire types. Mirrors github.com/NimbleMarkets/ntcharts/v2/spec
// (branch `spec`). Field names are the Go structs' json tags. Optional fields
// use undefined (never null) so JSON.stringify omits them, matching omitzero.

export interface NtFormat {
  kind?: "number" | "percent" | "currency" | "si" | "time";
  precision?: number;
  currency?: string;
  layout?: string; // Go time layout
}

export interface NtXAxis {
  title?: string;
  type?: "category" | "time" | "value";
  labels?: string[];
  format?: NtFormat;
}

export interface NtYAxis {
  title?: string;
  min?: number;
  max?: number;
  labels?: string[]; // added for heatmap row labels (Task 5)
  format?: NtFormat;
}

export interface NtDataPoint {
  x?: unknown; // string | number | RFC3339 string | ms-since-epoch
  y: number;
  size?: number;
}

export interface NtOHLCPoint { t: unknown; o: number; h: number; l: number; c: number; }

export interface NtSeries {
  name: string;
  type?: string;
  // Deliberate optionality exception: the Go struct's `Values []DataPoint`
  // field always serializes (Go's zero value for a nil slice marshals as
  // `null`/`[]`, never omitted), but this TS type marks `values` optional
  // so a series with no point data (e.g. an OHLC-only series, see `ohlc`
  // below) can omit the key entirely rather than emit `values: []`/`null`.
  // Producers that mean to emit an empty series should still set `[]`
  // explicitly; `undefined` here should be reserved for "not applicable".
  values?: NtDataPoint[];
  ohlc?: NtOHLCPoint[];
  color?: string;
}

export interface NtHeatCell { x: number; y: number; z: number; }
export interface NtHeatData {
  cells?: NtHeatCell[];
  matrix?: number[][];
  min_value?: number;
  max_value?: number;
}

export interface NtOptions {
  show_legend?: boolean;
  show_grid?: boolean;
  orientation?: "vertical" | "horizontal";
  stacked?: boolean;
  candle_style?: "line" | "block";
}

export interface NtTheme {
  background?: string;
  foreground?: string;
  palette?: string[];
  gradient?: string[];
}

export interface NtData { series: NtSeries[]; x_axis_data?: unknown[]; }

export interface NtSpec {
  type: "bar" | "line" | "timeseries" | "streamline" | "sparkline" | "heatmap" | "ohlc" | "scatter" | "canvas";
  title?: string;
  subtitle?: string;
  width: number;
  height: number;
  x_axis?: NtXAxis;
  y_axis?: NtYAxis;
  data: NtData;
  heat?: NtHeatData;
  options?: NtOptions;
  theme?: NtTheme;
}

import type { ChartWarning } from "flint-chart/core";
export type NtSpecOut = NtSpec & { _warnings?: ChartWarning[]; _width?: number; _height?: number };
