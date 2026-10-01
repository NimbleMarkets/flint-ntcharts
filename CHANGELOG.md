# CHANGELOG

## Unreleased

 * Opt-in raster renderer: `"renderer": "raster"` on the input (or `envelope.WithRenderer`) compiles to flint's ECharts option, and the new `raster` package draws it as an image with go-analyze. This adds chart types the text renderer cannot draw, such as grouped bars, pies and radars. `Render` returns `raster.ErrBlank` when the chart drew no data. New API: `envelope.ParseResult`, `envelope.Result`, `compile.Runner.CompileResult`. The embedded compiler grows from 1.9 MB to 3.4 MB.
 * `flint-tui` and `flint-edit` show raster charts with Kitty graphics or glyphs (`g` / `ctrl+g` to switch) and fall back to the text chart, with a `raster-fallback` warning, when raster cannot draw one. New API: `tui.RenderFrame`, `tui.Frame`, `tui.Pane`, `tui.ResultCompiler`. `flint-edit` gains a Grouped Bars (raster) example.
 * An unknown chart type now suggests the raster renderer in its error.

## v0.2.1 (2026-10-01)

 * `flint-edit` now runs in the browser: <https://nimblemarkets.github.io/flint-ntcharts/>. It is built from `demo/` and deployed to GitHub Pages on every push to `main`. `task site` builds it; `task site-serve` serves it locally.
 * No library or API changes since v0.2.0.

## v0.2.0 (2026-10-01)

 * Requires ntcharts v2.6.0.
 * New chart types: ECDF Plot, Connected Scatter Plot, Histogram, Calendar Heatmap, and three approximations that report an `info` warning: Bubble Chart (a scatter plot, size not shown), Area Chart (a line chart, no fill) and Lollipop Chart (a bar chart).
 * `logScale_x` / `logScale_y` draw a logarithmic axis on line, scatter, time-series and candlestick charts. Charts or data a log axis cannot show still warn and stay linear.
 * Line, scatter and sparkline charts, and heatmaps with numeric-looking column names, fill the size they are asked for instead of about 45% of the width. **Behavior change:** the compiled spec's `width` / `height` for those charts are larger.
 * Fix bar charts with many categories rendering as an empty axis: they are truncated to the bars the terminal can draw, with an `overflow` warning.
 * `includeZero_y` is honoured on line, time-series and scatter charts. Properties that cannot be honoured (`includeZero_x`, `includeZero_y` on bars) now produce a warning instead of being ignored silently.
 * `flint-edit` gains Log Scale, Heatmap, Histogram and Calendar Heatmap examples.

## v0.1.0 (2026-09-30)

First release.

 * Compile [flint-chart](https://github.com/microsoft/flint-chart) specs to [ntcharts](https://github.com/NimbleMarkets/ntcharts) terminal charts. The compiler is embedded as WebAssembly; no Node.js at run time.
 * Go API: `compile.New`, `Compile`, `CompileRaw`.
 * Chart types: bar, stacked bar, line, time series, scatter, heatmap, candlestick, sparkline.
 * `flint-tui`: live chart window fed by a file, stdin, or a unix socket.
 * `flint-edit`: split-pane playground — edit a spec, watch the chart update.
 * Browser build of the compiler for booba-shim.
 * Requires Go 1.26.8+. Built on flint-chart 0.5.1 and ntcharts v2.5.0.
