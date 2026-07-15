import { readFileSync, writeFileSync, mkdirSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { compileToVegaLite } from "../src/compile.js";

const fixturesDir = fileURLToPath(new URL("../../testdata/fixtures/", import.meta.url));
const expectedDir = fileURLToPath(new URL("../../testdata/expected/", import.meta.url));
mkdirSync(expectedDir, { recursive: true });

let failed = 0;
for (const f of readdirSync(fixturesDir).sort()) {
  if (!f.endsWith(".json")) continue;
  try {
    const out = compileToVegaLite(readFileSync(fixturesDir + f, "utf8"));
    writeFileSync(expectedDir + f, out);
    console.log(`ok   ${f} (${out.length} bytes)`);
  } catch (err) {
    failed++;
    console.error(`FAIL ${f}: ${err.message}`);
  }
}
process.exit(failed ? 1 : 0);
