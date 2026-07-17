import { compileToNtSpec } from "./compile.js";

// Injected by esbuild --define at build time (see package.json build:browser
// script). When this module is imported directly by source (vitest, tsx —
// no esbuild define pass applied), the identifier is left undefined by the
// bundler/runtime, so we fall back to a dev marker.
declare const __FLINTCHART_VERSION__: string;
const VERSION =
  typeof __FLINTCHART_VERSION__ !== "undefined" ? __FLINTCHART_VERSION__ : "dev";

// Registration contract frozen for the Go bridge (booba-shim vendors this
// bundle standalone — no bundler, no DOM). Must use globalThis, not window,
// so the bundle also loads in Node (vitest parity test, node-smoke.mjs).
declare global {
  // eslint-disable-next-line no-var
  var boobaShim: Record<string, unknown> | undefined;
}

const NAMESPACE: Record<string, unknown> = (globalThis.boobaShim = globalThis.boobaShim ?? {});

// compile is synchronous: flint compilation is pure computation. It either
// returns an envelope string ({"spec"...} or {"error"...}) or throws only on
// catastrophic engine failure — compileToNtSpec already catches JS errors
// into the error envelope.
NAMESPACE.flintchart = {
  compile: (inputJSON: string): string => compileToNtSpec(inputJSON),
  version: `flint-ntcharts ${VERSION}`,
  ready: Promise.resolve(),
};
