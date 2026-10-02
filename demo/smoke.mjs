// smoke.mjs — drives the real flint-edit wasm headlessly through the same
// bubbletea_* globals the page's terminal adapter uses: size it, read the
// screen, send keys, read again. Run from this directory after `task site`
// has built dist/ (app.wasm + shim assets):  node smoke.mjs
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";

const web = new URL("./dist/", import.meta.url);
createRequire(import.meta.url)("./dist/wasm_exec.js");
await import(new URL("booba-shim/flintchart/flintchart-shim.js", web));

const go = new Go();
const { instance } = await WebAssembly.instantiate(readFileSync(new URL("app.wasm", web)), go.importObject);
go.run(instance); // blocks inside the Go program; do not await

const strip = (s) => s.replace(/\x1b\[[0-9;?]*[ -\/]*[@-~]|\x1b\][^\x07]*\x07/g, "");
let screen = "";
const drain = () => { const d = globalThis.bubbletea_read(); if (d) screen += d; };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const fail = (msg) => { console.error("FAIL:", msg, "\n--- screen tail ---\n" + strip(screen).slice(-600)); process.exit(1); };

// waitFor polls the program's output until re matches or the timeout passes,
// so the test is as fast as the machine allows and does not flake under load.
const waitFor = async (re, what, timeoutMs = 15000) => {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    drain();
    if (re.test(strip(screen))) { console.log("ok  ", what); return; }
    if (Date.now() > deadline) fail(`expected ${what}`);
    await sleep(50);
  }
};

await sleep(300);
globalThis.bubbletea_resize(110, 30);
await waitFor(/compiled ok/, "first example compiles");
await waitFor(/Jan[\s\S]*Feb[\s\S]*Mar/, "bar chart category labels drawn");

screen = "";
globalThis.bubbletea_write("\x0e"); // ctrl+n
await waitFor(/price/, "ctrl+n loads the next example (its spec shows in the editor)");
// The status line may not be redrawn (the renderer only writes changed cells),
// so check the new chart itself: the time series has dollar Y labels.
await waitFor(/\$1[0-2]\d/, "the time series chart is drawn with price labels");

// Raster charts need a shim bundle built from flint-ntcharts v0.3.0 or later.
// An older vendored bundle (the booba-shim release in go.mod) answers
// "renderer": "raster" with an error, so only check the image when the bundle
// can produce one.
const probe = JSON.parse(globalThis.boobaShim.flintchart.compile(JSON.stringify({
  renderer: "raster",
  data: { values: [{ a: "x", b: 1 }] },
  chart_spec: { chartType: "Bar Chart", encodings: { x: { field: "a" }, y: { field: "b" } } },
})));
if (probe.echarts) {
  // examples 2 -> 8; one at a time so a slow render cannot swallow a keypress
  for (let i = 0; i < 6; i++) {
    screen = "";
    globalThis.bubbletea_write("\x0e");
    await sleep(250);
    drain();
  }
  // Without a Kitty reply the pane draws the image as half-block glyphs: whole
  // rows of lower half blocks, which no text chart produces.
  await waitFor(/▄{30}/, "the raster example draws an image", 30000);
  if (/Unknown chart type/.test(strip(screen))) fail("the raster example fell back to a text error");
} else {
  console.log("skip  raster example: the vendored shim bundle predates raster support");
}

screen = "";
globalThis.bubbletea_write("x");
await waitFor(/invalid JSON/, "a bad edit reports invalid JSON on the status line");

console.log("smoke ok");
process.exit(0);
