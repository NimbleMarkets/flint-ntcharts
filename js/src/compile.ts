import { assembleNtcharts } from "./ntcharts/index.js";

// Single shared compile path for Node reference runs and the Javy wasm build.
// Emits the frozen envelope contract: { spec, warnings, size } — see
// docs/2026-07-16-wasm-rewire-plan.md Global Constraints.
export function compileToNtSpec(inputJSON: string): string {
  const out = assembleNtcharts(JSON.parse(inputJSON));
  const { _warnings, _width, _height, ...spec } = out;
  return JSON.stringify({
    spec,
    warnings: _warnings ?? [],
    size: { width: _width, height: _height },
  });
}
