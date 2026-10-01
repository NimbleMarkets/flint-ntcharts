// Package raster draws ECharts options as images, using go-analyze/charts.
//
// flint's ECharts backend can describe far more chart types than the ntcharts
// text renderer draws (grouped bars, pies, radars, ...). Render turns one of
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

	"github.com/go-analyze/charts"
)

// ErrBlank is returned when the chart rendered without error but drew no data.
var ErrBlank = errors.New("raster: chart rendered blank")

// Render draws the ECharts option (as flint emits it) at w×h pixels. It
// returns [ErrBlank] when the chart's data drew nothing.
func Render(option []byte, w, h int) (image.Image, error) {
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
	p, err := charts.Render(eo.ToOption(), charts.DimensionsOptionFunc(w, h), charts.PNGOutputOptionFunc())
	if err != nil {
		return nil, fmt.Errorf("raster: render: %w", err)
	}
	return p.Bytes()
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
