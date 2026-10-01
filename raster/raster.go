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

// Render draws the ECharts option (as flint emits it) at w×h pixels.
func Render(option []byte, w, h int) (image.Image, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("raster: bad size %dx%d", w, h)
	}
	buf, err := renderPNG(Sanitize(option), w, h)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("raster: decode PNG: %w", err)
	}
	if !hasInk(img) {
		return nil, ErrBlank
	}
	return img, nil
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

// hasInk reports whether the image holds data colour. Axes, grid lines and
// labels are gray; series are drawn in saturated colours, so a chart with
// none of them drew nothing. A handful of pixels is not data (a stray mark),
// so a small fraction of the image is required. Measured on flint's own
// charts, sparse scatter plots reach 0.0016 and charts that drew nothing are
// exactly 0, so the bar sits well below the first and above the second.
func hasInk(img image.Image) bool {
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
	return sampled > 0 && float64(saturated)/float64(sampled) >= 0.0003
}

func chroma(r, g, b uint32) uint32 {
	hi, lo := max(r, g, b), min(r, g, b)
	return hi - lo
}
