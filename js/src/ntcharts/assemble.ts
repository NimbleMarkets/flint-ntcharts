import type { ChartAssemblyInput } from "flint-chart";
import type { NtSpecOut } from "./types.js";
import { ntGetTemplateDef, ntSupportedChartTypes } from "./templates/index.js";

export function assembleNtcharts(input: ChartAssemblyInput): NtSpecOut {
  const chartType = input.chart_spec.chartType;
  const def = ntGetTemplateDef(chartType);
  if (!def) {
    throw new Error(
      `Unknown chart type "${chartType}". Supported: ${ntSupportedChartTypes().join(", ")}`,
    );
  }
  throw new Error("assembleNtcharts: pipeline not implemented yet (Task 3)");
}
