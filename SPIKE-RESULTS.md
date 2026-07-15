# Javy WASM Spike Results (2026-07-15)

**Verdict: GO** — 5/5 fixtures byte-identical to Node reference, and per-compile latency (8.25 ms) is over 30x under the 250 ms gate. The one correctness blocker found during the spike (`structuredClone` trap) was root-caused and fixed with a small polyfill; no other divergence was observed.

| Metric | Value | Gate |
| --- | --- | --- |
| Fixtures parity-passing | 5 / 5 | 5 / 5 |
| Per-compile latency | 8.25 ms (8,250,012 ns/op, mean of 10 runs) | < 250 ms |
| flint.wasm size | 3,292,520 bytes (≈ 3.14 MB) | informational |
| JS bundle size | 710,901 bytes (≈ 694.2 KB per esbuild's own report) | informational |
| Javy version | v9.0.0 | — |
| QuickJS-risk audit hits (Intl/toLocale/etc.) | Intl. : 0; structuredClone: 2 (fixed, see below); toLocaleString: 5; toLocaleDateString: 4; toLocaleTimeString: 0; WeakRef: 0; FinalizationRegistry: 0 | informational |

## Benchmark raw output

```
$ go test ./compile/ -bench=. -benchtime=10x -run=^$ -v
goos: darwin
goarch: arm64
pkg: github.com/NimbleMarkets/flint-ntcharts/compile
cpu: Apple M3 Max
BenchmarkCompileScatter
BenchmarkCompileScatter-16    	      10	   8250012 ns/op
PASS
ok  	github.com/NimbleMarkets/flint-ntcharts/compile	1.914s
```

8,250,012 ns/op = 8.250012 ms per `CompileVegaLite` call (scatter fixture, 10 iterations, single-threaded, warm `Runner`/compiled module, fresh instantiation per call as `Runner` is designed for concurrent use).

## Divergences found

One divergence was found and fixed during the spike (Task 4); zero divergences remain in the current artifact.

- **`structuredClone` trap (found, fixed).** The bundle calls `structuredClone` twice inside flint-chart's own compiled code (`js/dist/flint-javy.js:1637` in `assembleVegaLite`'s data-copy path, and `:8488` in `assembleVegaLite`'s template copy). `structuredClone` is a Web/Node platform global, not part of the ECMAScript spec QuickJS implements, so `javy build` succeeded but every one of the 5 fixtures trapped at runtime with `Error: structuredClone is not defined` (WASI exit via `unreachable`, confirmed via `wasmtime` in Task 4). Fix: `js/src/polyfills.js` installs a guarded (`typeof globalThis.structuredClone !== "function"`) recursive deep-clone polyfill (handles `null`/primitives, `Date`, `Array`, `Map`, `Set`, `RegExp`, plain objects) imported first in `js/src/entry-javy.js`, so Node's native `structuredClone` is untouched and only QuickJS gets the shim. After the fix, all 5 fixtures ran to completion (exit 0, empty stderr) and were byte-identical to Node reference output both via direct `wasmtime` invocation (Task 4) and via the Go `TestParityWithNode` suite (Task 5) — this benchmark run used the same fixed artifact.
- **`toLocaleString`/`toLocaleDateString` (flagged as risk, no divergence observed).** 9 call sites were flagged in the Task 3 audit as locale-sensitive formatting that QuickJS might not implement identically (QuickJS has 0 `Intl.` hits, so full ICU-backed locale formatting is not available). Despite the flag, all 5 fixtures — including `bar-currency.json` and `line-temporal.json`, the ones most likely to exercise these formatters — produced byte-identical output to Node in both the Task 4 `wasmtime` check and the Task 5/6 Go parity test. No divergence found for the locale/format inputs exercised by the current fixture set.

## Notes for Phase 2+

- **The `structuredClone` polyfill is a shim, not a spec-complete implementation, and should be hardened or reconsidered before Phase 2 relies on it further.** It (`js/src/polyfills.js`) does not handle cycles (a self-referential object clone will stack-overflow/infinite-loop rather than erroring cleanly like the native API), and it "fails open" on exotic types it doesn't special-case (functions, symbols, typed arrays/`ArrayBuffer`, `Error` objects, class instances with private state, cross-realm objects) by falling through to plain-object copying, which can silently produce structurally-wrong clones rather than throwing. It happens to be sufficient for the plain chart-spec data (objects/arrays/primitives/Dates) the current 5 fixtures exercise, but new chart types or data shapes should be checked against it explicitly rather than assumed safe.
- **The QuickJS/`Intl`-less `toLocaleString`/`toLocaleDateString` risk is unresolved, not disproven — it just didn't trigger with these 5 fixtures.** All fixtures use `"en-US"` or default locale calls; the audit found no true multi-locale exercise. Before broadening the fixture/chart-type coverage in Phase 2, add fixtures that stress non-`en-US` locales and more exotic `Intl.NumberFormat`/`DateFormat` options to confirm the coincidental match holds generally, since QuickJS has zero native `Intl` support.
- **The Go module floor is `go 1.25.0`, above the design spec's stated `1.23` floor**, because `github.com/tetratelabs/wazero v1.12.0` (the sole runtime dependency) requires it. This is a real constraint for any consumer targeting an older Go toolchain and should be reflected in the Phase 2 design doc / dependency matrix rather than left as spike trivia.
- **Latency was only measured single-threaded and only for one fixture (`scatter.json`, one of the simpler specs).** `Runner.CompileVegaLite` is designed to instantiate a fresh module per call specifically to be concurrency-safe, but this spike did not benchmark concurrent throughput or measure latency for the larger/more complex fixtures (e.g., `heatmap.json`, `stacked-bar.json`). Given the very large margin under the 250 ms gate (8.25 ms measured vs. 250 ms gate), this is unlikely to change the go/no-go call, but Phase 2 load-testing should confirm concurrent-instantiation overhead and per-fixture-size variance before committing to production latency SLOs.
- **`flint.wasm` (3.14 MB) and the JS bundle (694.2 KB) are both informational-only in this spike** — no size gate was specified, and neither is large enough on its own to be a concern for typical WASI-module distribution, but both should be tracked over time as flint-chart and its dependencies grow.
