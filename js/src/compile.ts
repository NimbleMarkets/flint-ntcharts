import { assembleECharts } from "flint-chart/echarts";
import { assembleNtcharts } from "./ntcharts/index.js";

// Single shared compile path for Node reference runs and the Javy wasm build.
// Emits the frozen envelope contract: { spec, warnings, size } — see
// docs/2026-07-16-wasm-rewire-plan.md Global Constraints.
//
// Error contract: on failure this returns (never throws) a JSON string
// `{"error":{"message":<string>}}` instead of the success envelope. Success
// vs failure is discriminated by the top-level key (`spec` vs `error`), not
// by catching an exception — this keeps the wasm entry point (which has no
// place to surface a thrown JS error) and the Node reference runner on the
// exact same code path.
export function compileToNtSpec(inputJSON: string): string {
  try {
    const input = JSON.parse(inputJSON);
    // A top-level `renderer` key picks the output: "text" (the default) is the
    // ntcharts-spec envelope above; "raster" is flint's ECharts option, for the
    // Go side to draw as an image. See compileToECharts.
    const renderer = input.renderer ?? "text";
    if (renderer === "raster") return compileToECharts(input);
    if (renderer !== "text") {
      throw new Error(`unknown renderer ${JSON.stringify(renderer)}: use "text" (default) or "raster"`);
    }
    const out = assembleNtcharts(input);
    const { _warnings, _width, _height } = out;
    if (_width === undefined || _height === undefined) {
      // Size must always be present in the envelope; a template that forgot
      // to set it is a backend bug, not a recoverable condition.
      throw new Error("assembleNtcharts output is missing _width/_height (size must always be present)");
    }
    // Strip ALL private (_-prefixed) keys generically, not just the three
    // known ones above — future templates may add their own _-scratch
    // fields without needing this function updated.
    const spec = Object.fromEntries(
      Object.entries(out).filter(([key]) => !key.startsWith("_")),
    );
    return JSON.stringify({
      spec,
      warnings: _warnings ?? [],
      size: { width: _width, height: _height },
    });
  } catch (err) {
    return JSON.stringify({
      error: { message: String(err instanceof Error ? err.message : err) },
    });
  }
}

// compileToECharts runs flint's own ECharts backend and returns
// { echarts, warnings, size } in place of { spec, ... }. The option is flint's
// output unchanged apart from its private (_-prefixed) scratch keys; reshaping
// it for a particular renderer is the renderer's business (see the Go raster
// package). size is the requested baseSize, in terminal cells, or the option's
// own size when none was given.
function compileToECharts(input: any): string {
  const { renderer: _renderer, ...assemblyInput } = input;
  const out: any = assembleECharts(assemblyInput);
  const echarts = Object.fromEntries(Object.entries(out).filter(([key]) => !key.startsWith("_")));
  const base = assemblyInput.chart_spec?.baseSize;
  const cells = (n: number) => Math.max(1, Math.round(n));
  const size = base
    ? { width: base.width, height: base.height }
    : { width: cells(out._width), height: cells(out._height) };
  return JSON.stringify({ echarts, warnings: out._warnings ?? [], size });
}
