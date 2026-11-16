package raster

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// go-analyze reads only the first number of a data point and places every
// series on evenly spaced category slots. flint emits [x, y] points on a value
// or time X axis for line, scatter, density, ECDF and regression charts, so
// drawn as they are their X is lost: each curve becomes its X values in data
// order. reshapeContinuousX puts those series on a grid of evenly spaced
// slots across the axis (flint's min/max when it set them): a line takes its
// interpolated value at each slot inside its own range, and a scatter point
// takes its nearest slot, or the nearest free one within a few slots when two
// points share an X. The axis becomes a category axis labelled with the slot
// positions.
const gridSlots = 360

type xyPoint struct{ x, y float64 }

func reshapeContinuousX(o map[string]any) {
	ax := soleAxis(o["xAxis"])
	if ax == nil {
		return
	}
	kind, _ := ax["type"].(string)
	if kind != "value" && kind != "time" {
		return
	}
	// A category Y (ranged dots, horizontal bars) is not an X-Y plot.
	if y := soleAxis(o["yAxis"]); y != nil {
		if t, _ := y["type"].(string); t == "category" {
			return
		}
	}
	series, _ := o["series"].([]any)
	points := make([][]xyPoint, len(series))
	lo, hi := math.Inf(1), math.Inf(-1)
	for i, s := range series {
		sm, ok := s.(map[string]any)
		if !ok || (sm["type"] != "line" && sm["type"] != "scatter") {
			continue
		}
		data, _ := sm["data"].([]any)
		for _, e := range data {
			if p, ok := xyOf(e); ok {
				points[i] = append(points[i], p)
				lo, hi = math.Min(lo, p.x), math.Max(hi, p.x)
			}
		}
	}
	if math.IsInf(lo, 1) {
		return
	}
	if v, ok := ax["min"].(float64); ok && v < lo {
		lo = v
	}
	if v, ok := ax["max"].(float64); ok && v > hi {
		hi = v
	}
	if hi == lo {
		lo, hi = lo-1, hi+1
	}
	// A numeric axis spans round bounds, and the grid has a slot on each
	// round step: go-analyze labels every (slots-1)/(ticks-1)-th slot, so
	// with splitNumber = ticks the labels read 20 40 60, not 20.65 35.43.
	slots, ticks := gridSlots, 0
	if kind == "value" {
		var step float64
		lo, hi, step = niceBounds(lo, hi)
		ticks = int(math.Round((hi-lo)/step)) + 1
		slots = (gridSlots-1)/(ticks-1)*(ticks-1) + 1
	}
	at := func(i int) float64 { return lo + (hi-lo)*float64(i)/float64(slots-1) }
	slotOf := func(x float64) int { return int(math.Round((x - lo) / (hi - lo) * float64(slots-1))) }

	for i, s := range series {
		if len(points[i]) == 0 {
			continue
		}
		sm := s.(map[string]any)
		grid := make([]any, slots)
		pts := points[i]
		sort.SliceStable(pts, func(a, b int) bool { return pts[a].x < pts[b].x })
		if sm["type"] == "line" {
			k := 0
			for g := range grid {
				x := at(g)
				if x < pts[0].x || x > pts[len(pts)-1].x {
					continue
				}
				for k+1 < len(pts) && pts[k+1].x < x {
					k++
				}
				a, b := pts[k], pts[min(k+1, len(pts)-1)]
				if b.x == a.x {
					grid[g] = a.y
				} else {
					grid[g] = a.y + (b.y-a.y)*(x-a.x)/(b.x-a.x)
				}
			}
		} else {
			for _, p := range pts {
				s := slotOf(p.x)
				for _, d := range []int{0, 1, -1, 2, -2, 3, -3} {
					if j := s + d; j >= 0 && j < slots && grid[j] == nil {
						grid[j] = p.y
						break
					}
				}
			}
		}
		sm["data"] = grid
	}

	labels := make([]any, slots)
	for g := range labels {
		if kind == "time" {
			labels[g] = timeLabel(at(g), hi-lo)
		} else {
			labels[g] = formatValue(at(g))
		}
	}
	boundaryGap := false
	ax["type"], ax["data"], ax["boundaryGap"] = "category", labels, boundaryGap
	delete(ax, "min")
	delete(ax, "max")
	if ticks > 0 {
		ax["splitNumber"] = float64(ticks)
	} else if _, ok := ax["splitNumber"]; !ok {
		ax["splitNumber"] = 6.0
	}
}

// soleAxis is the axis option of a chart with one axis of that kind.
func soleAxis(v any) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		return t
	case []any:
		if len(t) == 1 {
			m, _ := t[0].(map[string]any)
			return m
		}
	}
	return nil
}

// xyOf reads an [x, y, ...] point, or {value: [x, y, ...]}; x may be a date.
func xyOf(e any) (xyPoint, bool) {
	if m, ok := e.(map[string]any); ok {
		e = m["value"]
	}
	arr, ok := e.([]any)
	if !ok || len(arr) < 2 {
		return xyPoint{}, false
	}
	var x float64
	switch t := arr[0].(type) {
	case float64:
		x = t
	case string:
		ms, ok := parseDate(t)
		if !ok {
			return xyPoint{}, false
		}
		x = ms
	default:
		return xyPoint{}, false
	}
	y, ok := arr[1].(float64)
	return xyPoint{x, y}, ok && !math.IsNaN(x)
}

func timeLabel(ms, span float64) string {
	t := time.UnixMilli(int64(ms)).UTC()
	day := float64(24 * time.Hour / time.Millisecond)
	switch {
	case span >= 3*365*day:
		return t.Format("2006")
	case span >= 60*day:
		return t.Format("2006-01")
	case span >= 2*day:
		return t.Format("2006-01-02")
	default:
		return t.Format("01-02 15:04")
	}
}

// unstackLines draws stacked line series at their stacked heights: go-analyze
// draws each series at its own values, so flint's range area (an invisible
// base plus the band's height) came out as the low edge and the band's width.
// The base series is named for the edge it draws.
func unstackLines(o map[string]any) {
	series, _ := o["series"].([]any)
	// flint titles a range area's axis "low, high" and names an ungrouped band
	// after its low field; those two names are the edges'.
	var lowName, highName string
	if y := soleAxis(o["yAxis"]); y != nil {
		if title, _ := y["name"].(string); strings.Count(title, ", ") == 1 {
			parts := strings.SplitN(title, ", ", 2)
			lowName, highName = parts[0], parts[1]
		}
	}
	totals := map[string][]float64{}
	bands := map[string]string{}
	for _, s := range series {
		sm, ok := s.(map[string]any)
		stack, _ := sm["stack"].(string)
		if !ok || stack == "" || sm["type"] != "line" {
			continue
		}
		data, _ := sm["data"].([]any)
		sum := totals[stack]
		for len(sum) < len(data) {
			sum = append(sum, 0)
		}
		for i, e := range data {
			v, ok := scalarOf(e)
			if !ok {
				continue
			}
			sum[i] += v
			data[i] = sum[i]
		}
		totals[stack] = sum
		delete(sm, "stack")
		name, _ := sm["name"].(string)
		if len(name) >= 6 && name[:6] == "__base" {
			bands[stack] = name
		} else if base, ok := bands[stack]; ok {
			if name == lowName && highName != "" {
				renameSeries(series, base, lowName)
				sm["name"] = highName
			} else {
				renameSeries(series, base, name+" (low)")
				sm["name"] = name + " (high)"
			}
		}
	}
}

func renameSeries(series []any, from, to string) {
	for _, s := range series {
		if sm, ok := s.(map[string]any); ok && sm["name"] == from {
			sm["name"] = to
		}
	}
}

func scalarOf(e any) (float64, bool) {
	switch t := e.(type) {
	case float64:
		return t, true
	case map[string]any:
		return scalarOf(t["value"])
	}
	return 0, false
}

// themeRiverAsLines draws a streamgraph (ECharts themeRiver, which go-analyze
// does not draw) as one line per stream over its dates.
func themeRiverAsLines(o map[string]any) {
	series, _ := o["series"].([]any)
	if len(series) != 1 {
		return
	}
	sm, ok := series[0].(map[string]any)
	if !ok || sm["type"] != "themeRiver" {
		return
	}
	data, _ := sm["data"].([]any)
	var dates, names []string
	values := map[[2]string]float64{}
	seen := map[string]bool{}
	for _, e := range data {
		arr, ok := e.([]any)
		if !ok || len(arr) < 3 {
			continue
		}
		d, name := toString(arr[0]), toString(arr[2])
		v, _ := arr[1].(float64)
		if !seen["d"+d] {
			seen["d"+d] = true
			dates = append(dates, d)
		}
		if !seen["n"+name] {
			seen["n"+name] = true
			names = append(names, name)
		}
		values[[2]string{d, name}] += v
	}
	sort.SliceStable(dates, func(a, b int) bool {
		x, okA := parseDate(dates[a])
		y, okB := parseDate(dates[b])
		if okA && okB {
			return x < y
		}
		return dates[a] < dates[b]
	})
	lines := make([]any, len(names))
	for i, name := range names {
		vals := make([]any, len(dates))
		for j, d := range dates {
			if v, ok := values[[2]string{d, name}]; ok {
				vals[j] = v
			}
		}
		lines[i] = map[string]any{"type": "line", "name": name, "data": vals}
	}
	labels := make([]any, len(dates))
	for i, d := range dates {
		labels[i] = d
	}
	o["series"] = lines
	o["xAxis"] = map[string]any{"type": "category", "data": labels, "boundaryGap": false}
	o["yAxis"] = map[string]any{"type": "value"}
	legend, _ := o["legend"].(map[string]any)
	if legend == nil {
		legend = map[string]any{}
	}
	legendNames := make([]any, len(names))
	for i, n := range names {
		legendNames[i] = n
	}
	legend["data"] = legendNames
	o["legend"] = legend
	delete(o, "singleAxis")
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	}
	return ""
}

// pathOnContinuousX reports a line whose X goes back and forth (a connected
// scatter plot's path): go-analyze draws lines over X slots only, so it can
// draw a path neither as given nor sorted by X.
func pathOnContinuousX(option []byte) bool {
	var o map[string]any
	if json.Unmarshal(option, &o) != nil {
		return false
	}
	ax := soleAxis(o["xAxis"])
	if t, _ := ax["type"].(string); t != "value" && t != "time" {
		return false
	}
	series, _ := o["series"].([]any)
	for _, s := range series {
		sm, ok := s.(map[string]any)
		if !ok || sm["type"] != "line" {
			continue
		}
		data, _ := sm["data"].([]any)
		up, down := false, false
		var prev *float64
		for _, e := range data {
			p, ok := xyOf(e)
			if !ok {
				continue
			}
			if prev != nil {
				up = up || p.x > *prev
				down = down || p.x < *prev
			}
			x := p.x
			prev = &x
		}
		if up && down {
			return true
		}
	}
	return false
}

// pointsOnCategoryY reports a line or scatter series drawn against a category
// Y axis, which go-analyze cannot place.
func pointsOnCategoryY(option []byte) bool {
	var o map[string]any
	if json.Unmarshal(option, &o) != nil {
		return false
	}
	y := soleAxis(o["yAxis"])
	if t, _ := y["type"].(string); t != "category" {
		return false
	}
	series, _ := o["series"].([]any)
	for _, s := range series {
		if sm, ok := s.(map[string]any); ok && (sm["type"] == "line" || sm["type"] == "scatter") {
			return true
		}
	}
	return false
}

// includeZero gives a value Y axis the zero ECharts includes by default
// (scale unset or false) when flint set no range of its own: go-analyze
// ignores scale and fits the axis to the data, which can start a bar chart
// above its shortest bar and drop it.
func includeZero(o map[string]any) {
	var axes []map[string]any
	switch t := o["yAxis"].(type) {
	case map[string]any:
		axes = []map[string]any{t}
	case []any:
		for _, a := range t {
			if m, ok := a.(map[string]any); ok {
				axes = append(axes, m)
			}
		}
	}
	series, _ := o["series"].([]any)
	for i, ax := range axes {
		if t, _ := ax["type"].(string); t != "value" && t != "" {
			continue
		}
		if s, _ := ax["scale"].(bool); s {
			continue
		}
		if _, ok := ax["min"]; ok {
			continue
		}
		if _, ok := ax["max"]; ok {
			continue
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, s := range series {
			sm, ok := s.(map[string]any)
			if !ok {
				continue
			}
			if idx, _ := sm["yAxisIndex"].(float64); int(idx) != i {
				continue
			}
			data, _ := sm["data"].([]any)
			for _, e := range data {
				v, ok := scalarOf(e)
				if !ok {
					if arr, isArr := e.([]any); isArr && len(arr) > 0 {
						v, ok = arr[len(arr)-1].(float64)
					}
				}
				if ok {
					lo, hi = math.Min(lo, v), math.Max(hi, v)
				}
			}
		}
		switch {
		case math.IsInf(lo, 1):
		case lo >= 0:
			ax["min"] = 0.0
		case hi <= 0:
			ax["max"] = 0.0
		}
	}
}
