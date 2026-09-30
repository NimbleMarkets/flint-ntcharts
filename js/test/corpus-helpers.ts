import type { ChartAssemblyInput } from "flint-chart/core";
import { TEST_GENERATORS } from "flint-chart/test-data";
import { ntSupportedChartTypes } from "../src/ntcharts/templates/index.js";

// flint-chart ships its own test corpus (`flint-chart/test-data`): per chart
// type, a generator returning cases shaped for upstream's test harness, not
// for ChartAssemblyInput. This adapter turns one into the other so the same
// cases can be pushed through our backend -- a much wider net than the
// hand-written fixtures under testdata/, and the thing that catches upstream
// behaviour changes (scheme renames, overflow-order changes, new warnings)
// that the fixtures never exercise.

export interface CorpusCase {
  id: string;
  chartType: string;
  input: ChartAssemblyInput;
}

interface UpstreamCase {
  title: string;
  chartType: string;
  data: Record<string, unknown>[];
  encodingMap: Record<string, { fieldID: string } | undefined>;
  metadata: Record<string, { semanticType?: string } | undefined>;
}

export function toAssemblyInput(c: UpstreamCase): ChartAssemblyInput {
  const encodings: Record<string, { field: string }> = {};
  for (const [channel, enc] of Object.entries(c.encodingMap)) {
    if (enc?.fieldID) encodings[channel] = { field: enc.fieldID };
  }
  const semantic_types: Record<string, string> = {};
  for (const [field, meta] of Object.entries(c.metadata)) {
    if (meta?.semanticType) semantic_types[field] = meta.semanticType;
  }
  return {
    data: { values: c.data },
    semantic_types,
    chart_spec: { chartType: c.chartType, encodings },
  };
}

// Every upstream case for every chart type this backend registers. Types we
// don't support are skipped on purpose: the corpus is a regression net for
// what we claim to compile, not a coverage report.
export function supportedCorpus(): CorpusCase[] {
  const generators = TEST_GENERATORS as Record<string, (() => UpstreamCase[]) | undefined>;
  const out: CorpusCase[] = [];
  for (const chartType of ntSupportedChartTypes()) {
    const gen = generators[chartType];
    if (!gen) continue;
    for (const c of gen()) {
      out.push({ id: `${chartType} :: ${c.title}`, chartType, input: toAssemblyInput(c) });
    }
  }
  return out;
}
