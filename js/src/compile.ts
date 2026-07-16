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
    const out = assembleNtcharts(JSON.parse(inputJSON));
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
