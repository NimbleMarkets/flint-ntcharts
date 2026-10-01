# CHANGELOG

## Unreleased

 * Requires ntcharts v2.6.0 (log axes, labelled and filled heatmaps, connected numeric line charts).
 * `logScale_x` / `logScale_y` now draw a logarithmic axis on line, scatter, time-series and candlestick charts instead of warning; charts or data a log axis cannot show still warn and stay linear.
 * `flint-edit` gains Log Scale and Heatmap examples.
 * Fix bar charts with many categories rendering as an empty axis: charts are truncated to the bars the terminal can draw (half the width, or half the height for horizontal bars), with an `overflow` warning.
 * `includeZero_y` is honoured on line, time-series and scatter charts. Where a property cannot be honoured (`includeZero_x`, `includeZero_y` on bars, `logScale_x`/`logScale_y`) the envelope now carries a warning instead of ignoring it silently.

## v0.1.0 (2026-09-30)

First release.

 * Compile [flint-chart](https://github.com/microsoft/flint-chart) specs to [ntcharts](https://github.com/NimbleMarkets/ntcharts) terminal charts. The compiler is embedded as WebAssembly; no Node.js at run time.
 * Go API: `compile.New`, `Compile`, `CompileRaw`.
 * Chart types: bar, stacked bar, line, time series, scatter, heatmap, candlestick, sparkline.
 * `flint-tui`: live chart window fed by a file, stdin, or a unix socket.
 * `flint-edit`: split-pane playground — edit a spec, watch the chart update.
 * Browser build of the compiler for booba-shim.
 * Requires Go 1.26.8+. Built on flint-chart 0.5.1 and ntcharts v2.5.0.
