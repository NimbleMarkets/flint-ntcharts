import { readFileSync, writeFileSync, mkdirSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { compileToNtSpec } from "../src/compile.js";

const PAIRS = [
  ["../../testdata/fixtures/", "../../testdata/expected/"],
  ["../../testdata/fixtures-terminal/", "../../testdata/expected-terminal/"],
];

let failed = 0;
for (const [fixRel, expRel] of PAIRS) {
  const fixturesDir = fileURLToPath(new URL(fixRel, import.meta.url));
  const expectedDir = fileURLToPath(new URL(expRel, import.meta.url));
  mkdirSync(expectedDir, { recursive: true });
  for (const f of readdirSync(fixturesDir).sort()) {
    if (!f.endsWith(".json")) continue;
    try {
      const out = compileToNtSpec(readFileSync(fixturesDir + f, "utf8"));
      writeFileSync(expectedDir + f, out);
      console.log(`ok   ${expRel.split("/").at(-2)}/${f} (${Buffer.byteLength(out)} bytes)`);
    } catch (err) {
      failed++;
      console.error(`FAIL ${f}: ${err.message}`);
    }
  }
}
process.exit(failed ? 1 : 0);
