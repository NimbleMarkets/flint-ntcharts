// Verifies the built browser bundle loads and compiles in a bare JS runtime
// (what booba-shim vendors must work standalone — no bundler, no DOM).
import { readFileSync } from "node:fs";
await import("../dist/flintchart-shim.mjs");
const ns = globalThis.boobaShim?.flintchart;
if (!ns) { console.error("FAIL: namespace not registered"); process.exit(1); }
await ns.ready;
const input = readFileSync(new URL("../../testdata/fixtures/scatter.json", import.meta.url), "utf8");
const out = JSON.parse(ns.compile(input));
if (!out.spec || out.spec.type !== "scatter") { console.error("FAIL: bad envelope", out); process.exit(1); }
console.log(`ok flintchart-shim.mjs: scatter → ${out.size.width}x${out.size.height}, ${out.warnings.length} warning(s)`);
const dualInput = readFileSync(new URL("../../testdata/fixtures-terminal/combo.json", import.meta.url), "utf8");
const dual = JSON.parse(ns.compile(dualInput));
if (!dual.spec?.y2_axis || dual.spec.data.series[1]?.y_axis !== "right" || dual.warnings.length) {
  console.error("FAIL: dual-axis envelope", dual); process.exit(1);
}
console.log("ok flintchart-shim.mjs: combo chart");
