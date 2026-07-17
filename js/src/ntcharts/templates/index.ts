import type { ChartTemplateDef } from "flint-chart";
import { ntBarChartDef } from "./bar.js";
import { ntStackedBarChartDef } from "./stacked-bar.js";
import { ntLineChartDef } from "./line.js";
import { ntScatterPlotDef } from "./scatter.js";
import { ntHeatmapDef } from "./heatmap.js";
import { ntCandlestickChartDef } from "./candlestick.js";
import { ntSparklineDef } from "./sparkline.js";

const defs: ChartTemplateDef[] = [];

export function ntAllTemplateDefs(): ChartTemplateDef[] { return defs; }
export function ntGetTemplateDef(chart: string): ChartTemplateDef | undefined {
  return defs.find((d) => d.chart === chart);
}
export function ntSupportedChartTypes(): string[] { return defs.map((d) => d.chart); }
export function ntRegister(def: ChartTemplateDef): void { defs.push(def); }

ntRegister(ntBarChartDef);
ntRegister(ntStackedBarChartDef);
ntRegister(ntLineChartDef);
ntRegister(ntScatterPlotDef);
ntRegister(ntHeatmapDef);
ntRegister(ntCandlestickChartDef);
ntRegister(ntSparklineDef);
