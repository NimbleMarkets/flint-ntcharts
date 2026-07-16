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

import type { ChartWarning } from "flint-chart";
export type NtSpecOut = NtSpec & { _warnings?: ChartWarning[]; _width?: number; _height?: number };
