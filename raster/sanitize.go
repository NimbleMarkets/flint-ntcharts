package raster

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

// Sanitize reshapes the ECharts options flint emits into what go-analyze's
// ECharts parser accepts. It never fails: input it cannot parse is returned
// unchanged, so the caller sees go-analyze's own error.
//
// What it changes:
//   - data points: a leading date string becomes epoch milliseconds, a leading
//     category name is dropped, and trailing non-numeric values (a scatter
//     point's colour group) are cut, leaving the numbers go-analyze reads;
//   - pie radius: an [inner, outer] pair becomes its outer radius, and a bare
//     number becomes a string, as go-analyze expects;
//   - legend: a horizontal row at the top (go-analyze would draw flint's
//     right-hand column over the Y axis), position "middle" becomes
//     "center", and {name} entries become plain names;
//   - series on a value or time X axis are put on evenly spaced category
//     slots (go-analyze reads no X), stacked lines are drawn at their stacked
//     heights, and a themeRiver becomes one line per stream; see continuous.go;
//   - a value Y axis that should include zero (ECharts' default, which
//     go-analyze ignores) is given min 0 (or max 0) when flint set no range;
//   - sizes go-analyze reads as whole numbers (font sizes, margins, symbol
//     sizes): a fraction, as flint 1.1 emits for compact charts, is rounded.
func Sanitize(option []byte) []byte {
	var o map[string]any
	if json.Unmarshal(option, &o) != nil {
		return option
	}
	roundWholeFields(o)
	themeRiverAsLines(o)
	reshapeContinuousX(o)
	unstackLines(o)
	if ax := soleAxis(o["xAxis"]); ax != nil {
		// go-analyze reads category labels as strings only.
		if d, ok := ax["data"].([]any); ok {
			for i, v := range d {
				if f, ok := v.(float64); ok {
					d[i] = toString(f)
				}
			}
		}
	}
	if lg, ok := o["legend"].(map[string]any); ok {
		sanitizeLegend(lg)
		if t, ok := o["title"].(map[string]any); ok && lg["top"] != "bottom" {
			stackUnderTitle(t, lg)
		}
	} else {
		// ECharts draws no legend unless asked; go-analyze draws one from
		// the series names, unnamed series included.
		o["legend"] = map[string]any{"show": false}
	}
	series, _ := o["series"].([]any)
	for _, s := range series {
		if sm, ok := s.(map[string]any); ok {
			sanitizeSeries(sm)
		}
	}
	includeZero(o)
	out, err := json.Marshal(o)
	if err != nil {
		return option
	}
	return out
}

// stackUnderTitle puts a top legend a row below the title. go-analyze moves
// a legend under the title only when it sees the two overlap, which it misses
// for flint's numeric title offset: the legend stays at the edge and the
// title is drawn across it. The title goes to the top edge and the legend
// below it, a gap apart.
func stackUnderTitle(title, lg map[string]any) {
	if text, _ := title["text"].(string); text == "" {
		return
	}
	delete(title, "top")
	line := 23.0
	if ts, ok := title["textStyle"].(map[string]any); ok {
		if lh, ok := ts["lineHeight"].(float64); ok && lh > 0 {
			line = lh
		} else if fs, ok := ts["fontSize"].(float64); ok && fs > 0 {
			line = math.Round(fs * 1.3)
		}
	}
	lg["top"] = line + 12
}

func sanitizeLegend(lg map[string]any) {
	// go-analyze reads only a legend's left and top, so flint's vertical
	// legend at the right edge (right: N) falls to the left edge, over the Y
	// axis labels, and it reserves no room beside the plot. A horizontal row
	// at the top, which go-analyze keeps clear of the plot and wraps, reads
	// the same. A legend flint put at the bottom stays there: go-analyze
	// reads that as top "bottom", and would otherwise draw it at the top
	// edge, over the title.
	if b, _ := lg["bottom"]; b == nil {
		lg["orient"], lg["left"] = "horizontal", "center"
		delete(lg, "top")
		delete(lg, "right")
		delete(lg, "align")
	} else {
		lg["orient"], lg["top"] = "horizontal", "bottom"
		delete(lg, "bottom")
	}
	for _, k := range []string{"left", "right", "top", "bottom"} {
		if v, ok := lg[k].(string); ok && v == "middle" {
			lg[k] = "center"
		}
	}
	if d, ok := lg["data"].([]any); ok {
		for i, e := range d {
			if m, ok := e.(map[string]any); ok {
				if n, ok := m["name"].(string); ok {
					d[i] = n
				}
			}
		}
	}
}

func sanitizeSeries(s map[string]any) {
	switch r := s["radius"].(type) {
	case []any:
		if len(r) > 0 {
			s["radius"] = fmt.Sprint(r[len(r)-1])
		}
	case float64:
		s["radius"] = fmt.Sprint(r)
	}
	if d, ok := s["data"].([]any); ok {
		for i, e := range d {
			if arr, ok := e.([]any); ok {
				d[i] = numericValues(arr)
			}
		}
	}
}

// numericValues keeps the numbers of a data point. A string in the first slot
// is a date (converted) or a category name (dropped); a string anywhere later
// ends the point, since those are extra dimensions go-analyze cannot read.
func numericValues(point []any) []any {
	out := make([]any, 0, len(point))
	for i, v := range point {
		switch t := v.(type) {
		case float64, nil:
			out = append(out, t)
		case string:
			if i != 0 {
				return out
			}
			if ms, ok := parseDate(t); ok {
				out = append(out, ms)
			}
		default:
			return out
		}
	}
	return out
}

func parseDate(s string) (float64, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02", "2006-01", "2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return float64(t.UnixMilli()), true
		}
	}
	return 0, false
}

// wholeFields are the option keys go-analyze decodes into ints.
var wholeFields = map[string]bool{
	"fontSize": true, "splitNumber": true, "symbolSize": true, "distance": true, "yAxisIndex": true,
	"top": true, "bottom": true, "left": true, "right": true, "width": true, "height": true,
}

// roundWholeFields rounds fractional numbers under wholeFields keys anywhere
// in the option, leaving series data alone.
func roundWholeFields(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			if k == "data" {
				continue
			}
			if f, ok := e.(float64); ok && wholeFields[k] {
				t[k] = math.Round(f)
				continue
			}
			roundWholeFields(e)
		}
	case []any:
		for _, e := range t {
			roundWholeFields(e)
		}
	}
}
