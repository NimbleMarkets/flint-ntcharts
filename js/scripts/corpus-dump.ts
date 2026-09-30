// Dump the compiled envelope for every upstream test-data case we support,
// keyed by case id, so two flint-chart versions can be diffed:
//
//   npx tsx scripts/corpus-dump.ts before.json
//   npm install flint-chart@<new> && npx tsx scripts/corpus-dump.ts after.json
//   diff <(jq -S . before.json) <(jq -S . after.json)
//
// The vitest in test/corpus.test.ts asserts these all compile; this script
// exists to see *what changed* when they still do.
import { writeFileSync } from "node:fs";
import { compileToNtSpec } from "../src/compile.js";
import { supportedCorpus } from "../test/corpus-helpers.js";

const outPath = process.argv[2];
if (!outPath) {
  console.error("usage: tsx scripts/corpus-dump.ts <out.json>");
  process.exit(2);
}
const results: Record<string, unknown> = {};
let errors = 0;
for (const variant of ["pixel", "terminal"] as const) {
  for (const c of supportedCorpus()) {
    const input = structuredClone(c.input);
    if (variant === "terminal") input.chart_spec.baseSize = { width: 64, height: 20 };
    const out = JSON.parse(compileToNtSpec(JSON.stringify(input)));
    if (out.error) errors++;
    results[`${variant} :: ${c.id}`] = out;
  }
}
writeFileSync(outPath, JSON.stringify(results, null, 1) + "\n");
console.log(`${Object.keys(results).length} results, ${errors} error envelopes -> ${outPath}`);
