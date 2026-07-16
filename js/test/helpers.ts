import { readFileSync, writeFileSync, existsSync, mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { assembleNtcharts } from "../src/ntcharts/index.js";

const fixtureDir = fileURLToPath(new URL("../../testdata/fixtures/", import.meta.url));
const goldenDir = fileURLToPath(new URL("../../testdata/ntspec-golden/", import.meta.url));

export function assembleFixture(name: string) {
  const input = JSON.parse(readFileSync(`${fixtureDir}${name}.json`, "utf8"));
  return assembleNtcharts(input);
}

export function expectGolden(name: string, out: unknown): void {
  mkdirSync(goldenDir, { recursive: true });
  const path = `${goldenDir}${name}.json`;
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
