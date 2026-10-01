package raster

import (
	"encoding/json"
	"fmt"
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
//   - legend: position "middle" becomes "center", and {name} entries become
//     plain names.
func Sanitize(option []byte) []byte {
	var o map[string]any
	if json.Unmarshal(option, &o) != nil {
		return option
	}
	if lg, ok := o["legend"].(map[string]any); ok {
		sanitizeLegend(lg)
	}
	series, _ := o["series"].([]any)
	for _, s := range series {
		if sm, ok := s.(map[string]any); ok {
			sanitizeSeries(sm)
		}
	}
	out, err := json.Marshal(o)
	if err != nil {
		return option
	}
	return out
}

func sanitizeLegend(lg map[string]any) {
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
