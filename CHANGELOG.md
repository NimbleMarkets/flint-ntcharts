# CHANGELOG

## Unreleased

 * Requires flint-chart 1.1.0 and ntcharts v2.9.0 (both Go modules). The categorical line and scatter builder no longer needs a local ntcharts checkout.
 * Bars follow flint 1.1's banded axes: dates and numbers along the category axis are laid out as bands, in time order, named as briefly as their values allow (`2021`, `2021-03`, `2021-03-14`), and limited to what the width draws with an `overflow` warning.
 * Several rows on one bar combine by the measure's rule (sum, or average for intensive measures and `Number`) with a `rows-combined` warning, instead of the last row winning. A facet on a bar chart is merged into each bar with a `facets-merged` warning.
 * Bars below zero are drawn: the value axis is no longer pinned at 0 when values go negative.
 * Names too wide for their bars are shortened rather than cut (`'21`, `Mar`, `03`; a number that does not fit is left out), with a `labels-shortened` warning giving the axis range. Histogram bins are named by their starts with fewer figures where needed.
 * Scatter plots draw a categorical y beside a measured x transposed (`axes-transposed`), and dates along x on a time axis (they were dropped as NaN). Flattened facets get distinct colours.
 * Sparklines spread over the chart's width; several sparklines are drawn as one line chart (`chart-type-approximated`).
 * A numeric line chart whose data sits far from 0 says so (`x-axis-from-zero`): the terminal's line X axis always reaches 0.
 * **Behavior change:** some requests are declined with an error naming the fields instead of drawing an empty or merged chart: bars with no quantitative axis, a line chart whose y is not a measure, a scatter plot on two categorical axes, and a faceted candlestick chart.
 * New chart types from flint 1.1: Grouped Bar Chart (bar by bar, following flint's dodge plan), Pyramid Chart, Density Plot (flint's KDE), Range Area Chart (both edges), Streamgraph and Ranged Dot Plot (approximated), and Regression (flint's six fit methods, the fit drawn as a row of points). Charts that are approximated say so in an `info` warning. Pie, donut, rose, radar, funnel, treemap, sunburst, sankey, network, tree, parallel coordinates, gauge, bullet, boxplot, violin, waterfall, Gantt, bump, KPI and map charts remain raster-only or unsupported.
 * A line (and area, streamgraph or sparkline drawn as lines) whose dates run at a regular cadence with no gaps, with room for every label, is drawn on date slots in flint's date format, so each label sits under its point. ntcharts' time axis labels a fixed 4-column step, which put `Feb` up to three columns past Feb 1.
 * Raster: charts on a numeric or time X axis (line, scatter, density, ECDF, regression) keep their X; go-analyze had drawn each series' X values in data order. Range areas draw their low and high edges, streamgraphs draw one line per stream, points against a category Y axis and connected-scatter paths return `raster.ErrUnsupported`, so viewers fall back to text, legends are drawn as a row at the top (go-analyze had drawn flint's right-hand legend over the Y axis labels), and value axes that should include zero start at 0 (go-analyze had started a bar chart above its shortest bar).
 * flint-edit gains five examples: Grouped Bars, Pyramid, Density Plot, Range Area and Regression (14 in all). Examples may now carry `info` notes.
 * Raster: fractional sizes in flint 1.1's ECharts options (font sizes in compact charts) are rounded, so those charts render instead of failing to parse.

## v0.3.1 (2026-10-10)

 * Requires ntcharts v2.8.0 in both the main module and browser demo.
 * Native ntcharts specs can render independent left and right Y axes on line, scatter and time-series charts using `y2_axis` and `"y_axis": "right"`. Includes examples and regression tests across terminal sizes. The Flint compiler does not yet emit dual-axis specs.
 * When `g` / `ctrl+g` cannot switch an image to Kitty graphics, the status line now gives ntcharts' reason (probe still pending, terminal not recognized, or probe unanswered), which names the `NTCHARTS_KITTY=supported` override. `FLINT_KITTY` still works as an alias and applies only when `NTCHARTS_KITTY` is not set.
 * The browser demo uses booba-shim v0.4.0, which includes the raster-capable compiler bundle.

## v0.3.0 (2026-10-02)

 * Opt-in raster renderer. Add `"renderer": "raster"` to the input (or pass `envelope.WithRenderer("raster")`) and the compiler returns flint's ECharts option; the new `raster` package draws it as an image with go-analyze. This covers chart types the text renderer cannot draw, such as grouped bars, pies and radars. Where go-analyze draws nothing, `raster.Render` returns `raster.ErrBlank`.
 * `flint-tui` and `flint-edit` show raster charts as Kitty graphics or half-block glyphs, and fall back to the text chart with a `raster-fallback` warning when raster cannot draw one. `FLINT_KITTY=1` forces Kitty graphics where detection misses.
 * Keys: `r` in `flint-tui`, `ctrl+r` in `flint-edit`, flips a chart between the text and raster renderers; `g` / `ctrl+g` switches an image between Kitty graphics and glyphs.
 * `flint-edit` gains a Grouped Bars (raster) example, a `-example N` flag, and the example number and chart type in the lower-right corner.
 * An unknown chart type now suggests the raster renderer in its error.
 * New API: `envelope.ParseResult`, `envelope.Result`, `envelope.WithRenderer`, `compile.Runner.CompileResult`, `tui.RenderFrame`, `tui.FrameOptions`, `tui.Frame`, `tui.Pane`, `tui.ResultCompiler`, `tui.ToggleRenderer`. `tui.Render` is unchanged.
 * Cost: the embedded compiler grows from 1.9 MB to 3.4 MB, and the main module now depends on go-analyze.

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
