package raster

import (
	"errors"
	"strings"
	"testing"
)

func seriesData(t *testing.T, o map[string]any, i int) []any {
	t.Helper()
	return o["series"].([]any)[i].(map[string]any)["data"].([]any)
}

func TestContinuousX(t *testing.T) {
	t.Run("a line on a value axis keeps its X: interpolated on evenly spaced slots", func(t *testing.T) {
		in := `{"xAxis":{"type":"value"},"yAxis":{"type":"value"},"series":[
			{"type":"line","data":[[0,0],[10,10]]},
			{"type":"line","data":[[5,1],[10,1]]}]}`
		o := decode(t, Sanitize([]byte(in)))
		a, b := seriesData(t, o, 0), seriesData(t, o, 1)
		n := len(a)
		if n < gridSlots-10 || n > gridSlots || a[0] != 0.0 || a[n-1] != 10.0 {
			t.Fatalf("first line: %d slots, ends %v %v", n, a[0], a[n-1])
		}
		if mid := a[n/2].(float64); mid < 4.9 || mid > 5.1 {
			t.Fatalf("midpoint = %v, want about 5", mid)
		}
		if b[0] != nil || b[n-1] != 1.0 {
			t.Fatalf("a line starting at x=5 must leave the left half empty: %v … %v", b[0], b[n-1])
		}
		ax := o["xAxis"].(map[string]any)
		labels := ax["data"].([]any)
		if ax["type"] != "category" || len(labels) != n {
			t.Fatalf("xAxis = %v", ax["type"])
		}
		// go-analyze labels every (n-1)/(ticks-1)-th slot: each is a round value.
		ticks := int(ax["splitNumber"].(float64))
		var got []string
		for k := 0; k < ticks; k++ {
			got = append(got, labels[k*(n-1)/(ticks-1)].(string))
		}
		if strings.Join(got, " ") != "0 2 4 6 8 10" {
			t.Fatalf("labelled slots read %q, want 0 2 4 6 8 10", got)
		}
	})
	t.Run("scatter points sharing an X take neighbouring slots", func(t *testing.T) {
		in := `{"xAxis":{"type":"value","min":0,"max":100},"yAxis":{"type":"value"},"series":[
			{"type":"scatter","data":[[50,1],[50,2],[0,3]]}]}`
		d := seriesData(t, decode(t, Sanitize([]byte(in))), 0)
		var n int
		for _, v := range d {
			if v != nil {
				n++
			}
		}
		if n != 3 || d[0] != 3.0 {
			t.Fatalf("drew %d of 3 points; slot 0 = %v", n, d[0])
		}
	})
	t.Run("dates on a time axis are placed by time", func(t *testing.T) {
		in := `{"xAxis":{"type":"time"},"yAxis":{"type":"value"},"series":[
			{"type":"line","data":[["2024-01-01",1],["2024-01-02",1],["2024-12-31",3]]}]}`
		d := seriesData(t, decode(t, Sanitize([]byte(in))), 0)
		if v := d[gridSlots/2].(float64); v < 1.9 || v > 2.1 {
			t.Fatalf("mid-year value = %v: the second point must not take half the width", v)
		}
	})
}

func TestStackedLinesAreDrawnStacked(t *testing.T) {
	in := `{"xAxis":{"type":"category","data":["a","b"]},"series":[
		{"type":"line","name":"__base-0","stack":"band-0","data":[1,2]},
		{"type":"line","name":"Temp","stack":"band-0","data":[3,3]}]}`
	o := decode(t, Sanitize([]byte(in)))
	if d := seriesData(t, o, 1); d[0] != 4.0 || d[1] != 5.0 {
		t.Fatalf("high edge = %v, want [4 5]", d)
	}
	s := o["series"].([]any)
	if s[0].(map[string]any)["name"] != "Temp (low)" || s[1].(map[string]any)["name"] != "Temp (high)" {
		t.Fatalf("names = %v, %v", s[0].(map[string]any)["name"], s[1].(map[string]any)["name"])
	}
}

func TestUngroupedBandNamesItsEdges(t *testing.T) {
	in := `{"yAxis":{"type":"value","name":"Low °C, High °C"},"xAxis":{"type":"category","data":["a"]},"series":[
		{"type":"line","name":"__base-0","stack":"band-0","data":[1]},
		{"type":"line","name":"Low °C","stack":"band-0","data":[3]}]}`
	s := decode(t, Sanitize([]byte(in)))["series"].([]any)
	if s[0].(map[string]any)["name"] != "Low °C" || s[1].(map[string]any)["name"] != "High °C" {
		t.Fatalf("names = %v, %v", s[0].(map[string]any)["name"], s[1].(map[string]any)["name"])
	}
}

func TestThemeRiverBecomesLines(t *testing.T) {
	in := `{"singleAxis":{"type":"time"},"series":[{"type":"themeRiver","data":[
		["2024-02-01",5,"A"],["2024-01-01",1,"A"],["2024-01-01",2,"B"]]}]}`
	o := decode(t, Sanitize([]byte(in)))
	s := o["series"].([]any)
	if len(s) != 2 || s[0].(map[string]any)["type"] != "line" {
		t.Fatalf("series = %v", s)
	}
	if d := seriesData(t, o, 0); d[0] != 1.0 || d[1] != 5.0 {
		t.Fatalf("A = %v, want [1 5] in date order", d)
	}
	if d := seriesData(t, o, 1); d[1] != nil {
		t.Fatalf("B has no February value: %v", d)
	}
}

func TestRenderRefusesPointsOnACategoryY(t *testing.T) {
	in := `{"xAxis":{"type":"value"},"yAxis":{"type":"category","data":["a","b"]},"series":[{"type":"line","data":[[1,0],[3,0]]}]}`
	if _, err := Render([]byte(in), 320, 200); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
}

func TestNumericCategoryLabelsBecomeStrings(t *testing.T) {
	in := `{"xAxis":{"type":"category","data":[1,2.5]},"yAxis":{"type":"value"},"series":[{"type":"line","data":[3,4]}]}`
	d := decode(t, Sanitize([]byte(in)))["xAxis"].(map[string]any)["data"].([]any)
	if d[0] != "1" || d[1] != "2.5" {
		t.Fatalf("labels = %v", d)
	}
	if _, err := Render([]byte(in), 320, 200); err != nil {
		t.Fatalf("Render: %v", err)
	}
}

func TestValueAxisIncludesZero(t *testing.T) {
	in := `{"xAxis":{"type":"category","data":["a","b","c"]},"yAxis":{"type":"value","scale":false},"series":[{"type":"bar","data":[120,95,210]}]}`
	if y := decode(t, Sanitize([]byte(in)))["yAxis"].(map[string]any); y["min"] != 0.0 {
		t.Fatalf("yAxis.min = %v, want 0", y["min"])
	}
	fitted := `{"xAxis":{"type":"category","data":["a"]},"yAxis":{"type":"value","scale":true},"series":[{"type":"line","data":[120]}]}`
	if y := decode(t, Sanitize([]byte(fitted)))["yAxis"].(map[string]any); y["min"] != nil {
		t.Fatalf("a fitted axis (scale true) must keep its fit: min = %v", y["min"])
	}
}

func TestRenderRefusesAPathOnAContinuousX(t *testing.T) {
	path := `{"xAxis":{"type":"value"},"yAxis":{"type":"value"},"series":[{"type":"line","data":[[1,1],[3,2],[2,3]]}]}`
	if _, err := Render([]byte(path), 320, 200); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("err = %v, want ErrUnsupported", err)
	}
	line := `{"xAxis":{"type":"value"},"yAxis":{"type":"value"},"series":[{"type":"line","data":[[1,1],[2,3],[3,2]]}]}`
	if _, err := Render([]byte(line), 320, 200); err != nil {
		t.Fatalf("a line in X order must draw: %v", err)
	}
}

func TestLegendBecomesATopRow(t *testing.T) {
	in := `{"legend":{"orient":"vertical","right":10,"top":52,"align":"left","data":["a","b"]},"series":[]}`
	lg := decode(t, Sanitize([]byte(in)))["legend"].(map[string]any)
	if lg["orient"] != "horizontal" || lg["left"] != "center" || lg["top"] != nil || lg["right"] != nil {
		t.Fatalf("legend = %v", lg)
	}
}
