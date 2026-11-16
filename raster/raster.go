// Package raster draws ECharts options as images, using go-analyze/charts.
//
// flint's ECharts backend can describe far more chart types than the ntcharts
// text renderer draws (pies, radars, treemaps, ...). Render turns one of
// those options into an image that picture.Model can show with Kitty graphics
// or its glyph fallback.
//
// go-analyze renders only part of what flint emits, and some of what it
// cannot draw it draws as an empty plot without returning an error. Render
// reports that case as [ErrBlank], so callers can fall back to a text chart
// instead of showing nothing.
package raster

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"
	"strconv"
	"strings"

	"github.com/go-analyze/charts"
)

// ErrBlank is returned when the chart rendered without error but drew no data.
var ErrBlank = errors.New("raster: chart rendered blank")

// ErrUnsupported is returned for a chart go-analyze would draw wrongly: points
// or lines against a category Y axis (a ranged dot plot), which it would place
// by their index, or a line whose X goes back and forth (a connected scatter
// plot's path), which it can only draw over X slots.
var ErrUnsupported = errors.New("raster: go-analyze cannot draw this chart's points")

// Render draws the ECharts option (as flint emits it) at w×h pixels. It
// returns [ErrBlank] when the chart's data drew nothing.
func Render(option []byte, w, h int) (image.Image, error) {
	if pointsOnCategoryY(option) {
		return nil, fmt.Errorf("%w: points against a category Y axis", ErrUnsupported)
	}
	if pathOnContinuousX(option) {
		return nil, fmt.Errorf("%w: a path whose X goes back and forth", ErrUnsupported)
	}
	img, ink, err := render(Sanitize(option), w, h)
	if err != nil {
		return nil, err
	}
	if ink < inkThreshold {
		return nil, ErrBlank
	}
	return img, nil
}

// dataInk is the share of the image the chart's data draws in colour, for
// calibrating and testing the blank check.
func dataInk(option []byte, w, h int) (float64, error) {
	_, ink, err := render(option, w, h)
	return ink, err
}

// inkThreshold is the data-ink share below which a chart counts as blank.
// Sparse scatter plots measure about 0.0016 and charts that drew nothing
// measure 0 (see TestDataInkIgnoresTheLegend), so the bar sits between them.
const inkThreshold = 0.0003

func render(option []byte, w, h int) (image.Image, float64, error) {
	if w <= 0 || h <= 0 {
		return nil, 0, fmt.Errorf("raster: bad size %dx%d", w, h)
	}
	full, err := renderImage(option, w, h)
	if err != nil {
		return nil, 0, err
	}
	ink := saturatedShare(full)
	// Axes, grid lines and text are gray, but a legend's swatches are
	// saturated, so a chart whose series drew nothing can still show colour.
	// Subtract what the same chart shows with every series' data removed.
	if bare, ok := withoutData(option); ok {
		if empty, err := renderImage(bare, w, h); err == nil {
			ink -= saturatedShare(empty)
		}
	}
	return full, ink, nil
}

func renderImage(option []byte, w, h int) (image.Image, error) {
	buf, err := renderPNG(option, w, h)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("raster: decode PNG: %w", err)
	}
	return img, nil
}

// withoutData returns the option with every series' data emptied, keeping
// axes, legend and series names.
func withoutData(option []byte) ([]byte, bool) {
	var o map[string]any
	if json.Unmarshal(option, &o) != nil {
		return nil, false
	}
	series, _ := o["series"].([]any)
	for _, s := range series {
		if sm, ok := s.(map[string]any); ok {
			if _, has := sm["data"]; has {
				sm["data"] = []any{}
			}
		}
	}
	out, err := json.Marshal(o)
	return out, err == nil
}

func renderPNG(option []byte, w, h int) (buf []byte, err error) {
	// go-analyze panics on some option shapes; report them as errors.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("raster: render panic: %v", r)
		}
	}()
	var eo charts.EChartsOption
	if err := json.Unmarshal(option, &eo); err != nil {
		return nil, fmt.Errorf("raster: parse ECharts option: %w", err)
	}
	opt := eo.ToOption()
	if opt.ValueFormatter == nil {
		opt.ValueFormatter = formatValue
	}
	breathe(&opt)
	p, err := charts.Render(opt, charts.DimensionsOptionFunc(w, h), charts.PNGOutputOptionFunc())
	if err != nil {
		return nil, fmt.Errorf("raster: render: %w", err)
	}
	return p.Bytes()
}

// breathe gives a chart room between its parts. go-analyze ignores flint's
// grid margins and packs the plot to within a few pixels of the edges, with
// a value label every 25 pixels or so; at 640×360 that reads as crowded. The
// chart keeps a margin all round (more on the right, where the last X label
// overhangs the plot) and its value axes ask for fewer, rounder labels.
func breathe(opt *charts.ChartOption) {
	opt.Padding = charts.NewBox(16, 16, 28, 14)
	for i := range opt.YAxis {
		a := &opt.YAxis[i]
		if a.LabelCount != 0 || a.LabelCountAdjustment != 0 {
			continue
		}
		// A range flint pinned (a fitted domain) is split evenly between
		// its bounds, so labels read 3.06k, 3.57k, ...: round bounds and a
		// label on each round step instead, as flint's own nice scales do.
		if a.Min != nil && a.Max != nil && *a.Max > *a.Min {
			lo, hi, step := niceBounds(*a.Min, *a.Max)
			a.Min, a.Max = &lo, &hi
			a.LabelCount = int(math.Round((hi-lo)/step)) + 1
			no := false
			a.PreferNiceIntervals = &no
			continue
		}
		a.LabelCountAdjustment = -3
	}
	if x := &opt.XAxis; x.LabelCountAdjustment == 0 && len(opt.YAxis) > 0 && len(x.Labels) == 0 {
		x.LabelCountAdjustment = -2 // a horizontal bar chart's value axis
	}
}

// niceBounds widens lo..hi outward to multiples of a round step (1, 2 or 5
// times a power of ten) that splits it into about five intervals, and
// returns the step.
func niceBounds(lo, hi float64) (float64, float64, float64) {
	raw := (hi - lo) / 5
	p := math.Pow(10, math.Floor(math.Log10(raw)))
	step := 10 * p
	for _, m := range []float64{1, 2, 5} {
		if m*p >= raw {
			step = m * p
			break
		}
	}
	return math.Floor(lo/step+1e-9) * step, math.Ceil(hi/step-1e-9) * step, step
}

// formatValue formats axis and data values: as go-analyze does (two
// decimals, then k, M, ...) from 1 up, and with three significant digits
// below 1, where two decimals would print 0.012, 0.010 and 0.008 all as 0.01.
func formatValue(v float64) string {
	if a := math.Abs(v); a == 0 || a >= 1 {
		return charts.FormatValueHumanizeShort(v, 2, false)
	}
	s := strconv.FormatFloat(v, 'f', 2-int(math.Floor(math.Log10(math.Abs(v)))), 64)
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}

// saturatedShare is the fraction of (sampled) pixels with a saturated colour.
// Series are drawn in saturated colours; axes, grid lines and labels are gray.
func saturatedShare(img image.Image) float64 {
	b := img.Bounds()
	const stride = 2
	var sampled, saturated int
	for y := b.Min.Y; y < b.Max.Y; y += stride {
		for x := b.Min.X; x < b.Max.X; x += stride {
			r, g, bl, _ := img.At(x, y).RGBA()
			sampled++
			if chroma(r>>8, g>>8, bl>>8) > 40 {
				saturated++
			}
		}
	}
	if sampled == 0 {
		return 0
	}
	return float64(saturated) / float64(sampled)
}

func chroma(r, g, b uint32) uint32 {
	hi, lo := max(r, g, b), min(r, g, b)
	return hi - lo
}
