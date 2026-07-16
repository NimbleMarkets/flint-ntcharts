import type { ChartTemplateDef } from "flint-chart";

const defs: ChartTemplateDef[] = [];

export function ntAllTemplateDefs(): ChartTemplateDef[] { return defs; }
export function ntGetTemplateDef(chart: string): ChartTemplateDef | undefined {
  return defs.find((d) => d.chart === chart);
}
export function ntSupportedChartTypes(): string[] { return defs.map((d) => d.chart); }
export function ntRegister(def: ChartTemplateDef): void { defs.push(def); }
