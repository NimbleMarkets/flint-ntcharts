# CHANGELOG

## v0.1.0 (2026-09-30)

First release.

 * Compile [flint-chart](https://github.com/microsoft/flint-chart) specs to [ntcharts](https://github.com/NimbleMarkets/ntcharts) terminal charts. The compiler is embedded as WebAssembly; no Node.js at run time.
 * Go API: `compile.New`, `Compile`, `CompileRaw`.
 * Chart types: bar, stacked bar, line, time series, scatter, heatmap, candlestick, sparkline.
 * `flint-tui`: live chart window fed by a file, stdin, or a unix socket.
 * `flint-edit`: split-pane playground — edit a spec, watch the chart update.
 * Browser build of the compiler for booba-shim.
 * Requires Go 1.26.8+. Built on flint-chart 0.5.1 and ntcharts v2.5.0.
