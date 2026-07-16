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
      // compileToNtSpec no longer throws on compile failure -- it returns an
      // `{"error":{"message":...}}` envelope instead. An error envelope is a
      // SUCCESS only for a fixture meant to fail; all fixtures here are
      // meant to succeed, so treat one as a FAIL rather than committing it
      // as a reference (references must never be error envelopes).
      const parsed = JSON.parse(out);
      if (parsed && typeof parsed === "object" && "error" in parsed) {
        failed++;
        console.error(`FAIL ${f}: ${parsed.error?.message ?? "(no message)"}`);
        continue;
      }
      writeFileSync(expectedDir + f, out);
      console.log(`ok   ${expRel.split("/").at(-2)}/${f} (${Buffer.byteLength(out)} bytes)`);
    } catch (err) {
      failed++;
      console.error(`FAIL ${f}: ${err.message}`);
    }
  }
}
process.exit(failed ? 1 : 0);
