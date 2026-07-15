import { assembleVegaLite } from "flint-chart";

// Single shared code path for Node reference runs and the Javy wasm build.
// Byte parity between the two depends on both using exactly this function.
export function compileToVegaLite(inputJSON) {
  const input = JSON.parse(inputJSON);
  const spec = assembleVegaLite(input);
  return JSON.stringify(spec);
}
