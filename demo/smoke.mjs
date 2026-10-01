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
const settle = async (ms = 400) => { await new Promise((r) => setTimeout(r, ms)); drain(); };
const fail = (msg) => { console.error("FAIL:", msg, "\n--- screen tail ---\n" + strip(screen).slice(-600)); process.exit(1); };
const expect = (re, what) => { if (!re.test(strip(screen))) fail(`expected ${what}`); console.log("ok  ", what); };

await settle(300);
globalThis.bubbletea_resize(110, 30);
await settle(1500);
expect(/compiled ok/, "first example compiles");
expect(/Jan[\s\S]*Feb[\s\S]*Mar/, "bar chart category labels drawn");

screen = "";
globalThis.bubbletea_write("\x0e"); // ctrl+n
await settle(1500);
expect(/compiled ok/, "ctrl+n loads the next example cleanly");
expect(/Timeseries Line|price/, "the time series example is showing");

screen = "";
globalThis.bubbletea_write("x");
await settle(1000);
expect(/invalid JSON/, "a bad edit reports invalid JSON on the status line");

console.log("smoke ok");
process.exit(0);
