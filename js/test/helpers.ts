import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { assembleNtcharts } from "../src/ntcharts/index.js";

const fixtureDir = fileURLToPath(new URL("../../testdata/fixtures/", import.meta.url));
const goldenDir = fileURLToPath(new URL("../../testdata/ntspec-golden/", import.meta.url));

// dir param: "terminal" reads from testdata/fixtures-terminal/ and writes
// goldens to testdata/ntspec-golden-terminal/, keeping the pixel-scale and
// terminal-scale (no baseSize -> DEFAULT_TERMINAL_BASE) suites in separate
// fixture/golden directories.
const fixtureDirs: Record<string, string> = {
  default: fixtureDir,
  terminal: fileURLToPath(new URL("../../testdata/fixtures-terminal/", import.meta.url)),
};
const goldenDirs: Record<string, string> = {
  default: goldenDir,
  terminal: fileURLToPath(new URL("../../testdata/ntspec-golden-terminal/", import.meta.url)),
};

// Identity of a warning as a reader sees it. flint's filterOverflow returns
// the same overflow both in `warnings` and, with extra bookkeeping fields
// (keptValues, placeholder), in `truncations`; JSON equality would miss that.
export function warningKey(w: { code?: string; channel?: string; field?: string; message: string }): string {
  return [w.code, w.channel, w.field, w.message].join("|");
}

export function assembleFixture(name: string, dir: string = "default") {
  const base = fixtureDirs[dir];
  const input = JSON.parse(readFileSync(`${base}${name}.json`, "utf8"));
  return assembleNtcharts(input);
}

export function expectGolden(name: string, out: unknown, dir: string = "default"): void {
  const base = goldenDirs[dir];
  mkdirSync(base, { recursive: true });
  const path = `${base}${name}.json`;
  const rendered = JSON.stringify(out, null, 2) + "\n";
  if (process.env.UPDATE_GOLDEN === "1" || !existsSync(path)) {
    writeFileSync(path, rendered);
    return;
  }
  const expected = readFileSync(path, "utf8");
  if (rendered !== expected) {
    throw new Error(`golden mismatch for ${name}; run UPDATE_GOLDEN=1 to regenerate\n--- got ---\n${rendered.slice(0, 2000)}`);
  }
}
