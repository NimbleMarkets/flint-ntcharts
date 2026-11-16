package raster

import (
	"encoding/json"
	"errors"
	"github.com/go-analyze/charts"
	"os"
	"testing"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decode(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	return m
}

func TestSanitize(t *testing.T) {
	t.Run("trims scatter points to their numeric prefix", func(t *testing.T) {
		in := `{"series":[{"type":"scatter","data":[[1,2,"Low"],[3,4,"High"]]}]}`
		got := decode(t, Sanitize([]byte(in)))
		data := got["series"].([]any)[0].(map[string]any)["data"].([]any)
		for i, want := range [][2]float64{{1, 2}, {3, 4}} {
			p := data[i].([]any)
			if len(p) != 2 || p[0] != want[0] || p[1] != want[1] {
				t.Fatalf("point %d = %v, want %v", i, p, want)
			}
		}
	})
	t.Run("turns a leading date string into epoch milliseconds", func(t *testing.T) {
		in := `{"series":[{"type":"line","data":[["2026-01-01",5],["2026-02-01T00:00:00Z",6]]}]}`
		got := decode(t, Sanitize([]byte(in)))
		data := got["series"].([]any)[0].(map[string]any)["data"].([]any)
		if x := data[0].([]any)[0]; x != float64(1767225600000) {
			t.Fatalf("2026-01-01 -> %v", x)
		}
		if x := data[1].([]any)[0]; x != float64(1769904000000) {
			t.Fatalf("2026-02-01 -> %v", x)
		}
	})
	t.Run("pie radius array becomes its outer radius", func(t *testing.T) {
		in := `{"series":[{"type":"pie","radius":["40%","70%"],"data":[]}]}`
		got := decode(t, Sanitize([]byte(in)))
		if r := got["series"].([]any)[0].(map[string]any)["radius"]; r != "70%" {
			t.Fatalf("radius = %v", r)
		}
	})
	t.Run("pie numeric radius becomes a string", func(t *testing.T) {
		in := `{"series":[{"type":"pie","radius":80,"data":[]}]}`
		got := decode(t, Sanitize([]byte(in)))
		if r := got["series"].([]any)[0].(map[string]any)["radius"]; r != "80" {
			t.Fatalf("radius = %v", r)
		}
	})
	t.Run("legend position middle becomes center; named entries become names", func(t *testing.T) {
		in := `{"legend":{"left":"middle","data":[{"name":"a"},"b"]},"series":[]}`
		lg := decode(t, Sanitize([]byte(in)))["legend"].(map[string]any)
		if lg["left"] != "center" {
			t.Fatalf("left = %v", lg["left"])
		}
		if d := lg["data"].([]any); d[0] != "a" || d[1] != "b" {
			t.Fatalf("data = %v", d)
		}
	})
	t.Run("leaves input it cannot parse untouched", func(t *testing.T) {
		in := []byte(`{not json`)
		if got := Sanitize(in); string(got) != string(in) {
			t.Fatalf("changed: %s", got)
		}
	})
	t.Run("rounds the fractional sizes flint 1.1 emits, never data", func(t *testing.T) {
		in := `{"xAxis":[{"axisLabel":{"fontSize":6.5}}],"grid":{"left":40.4},"series":[{"type":"line","symbolSize":2.6,"data":[[1.5,2.5]]}]}`
		got := decode(t, Sanitize([]byte(in)))
		if f := got["xAxis"].([]any)[0].(map[string]any)["axisLabel"].(map[string]any)["fontSize"]; f != float64(7) {
			t.Fatalf("fontSize = %v", f)
		}
		if l := got["grid"].(map[string]any)["left"]; l != float64(40) {
			t.Fatalf("grid.left = %v", l)
		}
		s := got["series"].([]any)[0].(map[string]any)
		if s["symbolSize"] != float64(3) {
			t.Fatalf("symbolSize = %v", s["symbolSize"])
		}
		if p := s["data"].([]any)[0].([]any); p[0] != 1.5 || p[1] != 2.5 {
			t.Fatalf("data = %v", p)
		}
		if _, err := Render([]byte(`{"xAxis":[{"type":"category","data":["a","b"],"axisLabel":{"fontSize":6.5}}],"yAxis":[{"type":"value"}],"series":[{"type":"bar","data":[3,5]}]}`), 320, 200); err != nil {
			t.Fatalf("Render: %v", err)
		}
	})
	t.Run("drops a leading category name, keeping the values", func(t *testing.T) {
		in := `{"series":[{"type":"bar","data":[["Q1",5],["Q2",7]]}]}`
		data := decode(t, Sanitize([]byte(in)))["series"].([]any)[0].(map[string]any)["data"].([]any)
		for i, want := range []float64{5, 7} {
			if p := data[i].([]any); len(p) != 1 || p[0] != want {
				t.Fatalf("point %d = %v, want [%v]", i, p, want)
			}
		}
	})
}

func TestRenderDrawsCharts(t *testing.T) {
	for _, name := range []string{"grouped-bar", "pie", "dates-line", "scatter-size"} {
		t.Run(name, func(t *testing.T) {
			img, err := Render(fixture(t, name), 640, 400)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if b := img.Bounds(); b.Dx() != 640 || b.Dy() != 400 {
				t.Fatalf("size = %v, want 640x400", b)
			}
		})
	}
}

// go-analyze renders these without an error but draws an empty plot. Render
// must say so instead of handing back a blank image.
func TestRenderRefusesBlankCharts(t *testing.T) {
	for _, name := range []string{"waterfall", "waterfall-legend", "heatmap"} {
		t.Run(name, func(t *testing.T) {
			img, err := Render(fixture(t, name), 640, 400)
			if !errors.Is(err, ErrBlank) {
				t.Fatalf("err = %v, want ErrBlank", err)
			}
			if img != nil {
				t.Fatal("a blank render must not return an image")
			}
		})
	}
}

// The legend's colour swatches are saturated too. A chart whose only colour is
// its legend is still blank, and a chart with data is not blank just because
// the legend is large.
func TestDataInkIgnoresTheLegend(t *testing.T) {
	for name, wantInk := range map[string]bool{
		"waterfall-legend": false, // legend + axes, no bars
		"grouped-bar":      true,
		"scatter-size":     true, // sparse dots
		"pie":              true,
	} {
		ink, err := dataInk(Sanitize(fixture(t, name)), 640, 400)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		t.Logf("%-18s data ink %.5f", name, ink)
		if got := ink >= inkThreshold; got != wantInk {
			t.Errorf("%s: data ink %.5f -> ink=%v, want %v", name, ink, got, wantInk)
		}
	}
}

func TestRenderReportsBadOptions(t *testing.T) {
	if _, err := Render([]byte(`{nope`), 320, 200); err == nil || errors.Is(err, ErrBlank) {
		t.Fatalf("err = %v, want a parse error", err)
	}
}

func TestRenderRejectsDegenerateSizes(t *testing.T) {
	if _, err := Render(fixture(t, "grouped-bar"), 0, 0); err == nil {
		t.Fatal("expected an error for a zero size")
	}
}

func TestSmallAxisValuesStayDistinct(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i <= 6; i++ {
		s := formatValue(0.002 * float64(i))
		if seen[s] {
			t.Fatalf("%v prints %q, as a smaller tick does", 0.002*float64(i), s)
		}
		seen[s] = true
	}
	for v, want := range map[float64]string{0.012: "0.012", -0.0085: "-0.0085", 0.25: "0.25", 1500: "1.5k", 12.345: "12.35"} {
		if got := formatValue(v); got != want {
			t.Errorf("formatValue(%v) = %q, want %q", v, got, want)
		}
	}
}

func TestSanitizeStacksTheLegendUnderTheTitle(t *testing.T) {
	var o map[string]any
	in := `{"title":{"text":"T","top":15,"textStyle":{"fontSize":18,"lineHeight":23}},"legend":{"right":10,"orient":"vertical"},"series":[]}`
	if err := json.Unmarshal(Sanitize([]byte(in)), &o); err != nil {
		t.Fatal(err)
	}
	title, legend := o["title"].(map[string]any), o["legend"].(map[string]any)
	if _, ok := title["top"]; ok {
		t.Fatalf("title keeps its offset: %v", title)
	}
	if legend["top"] != 35.0 || legend["orient"] != "horizontal" {
		t.Fatalf("legend = %v, want a row at top 35 (a title line and a gap)", legend)
	}
}

func TestSanitizeHidesALegendFlintDidNotAskFor(t *testing.T) {
	var o map[string]any
	in := `{"series":[{"type":"scatter","data":[[1,2]]},{"type":"line","name":"Trend","data":[[1,2]]}]}`
	if err := json.Unmarshal(Sanitize([]byte(in)), &o); err != nil {
		t.Fatal(err)
	}
	if lg, _ := o["legend"].(map[string]any); lg["show"] != false {
		t.Fatalf("legend = %v, want hidden", o["legend"])
	}
}

func TestBreatheRoundsPinnedValueAxes(t *testing.T) {
	lo, hi := 3062.0, 6088.0
	opt := charts.ChartOption{YAxis: []charts.YAxisOption{{Min: &lo, Max: &hi}, {}}}
	breathe(&opt)
	pinned, free := opt.YAxis[0], opt.YAxis[1]
	if *pinned.Min != 3000 || *pinned.Max != 7000 || pinned.LabelCount != 5 {
		t.Fatalf("pinned axis = %v..%v with %d labels, want 3000..7000 every 1000", *pinned.Min, *pinned.Max, pinned.LabelCount)
	}
	if free.LabelCountAdjustment >= 0 {
		t.Fatalf("a free axis should ask for fewer labels, got %d", free.LabelCountAdjustment)
	}
	if opt.Padding.Top == 0 || opt.Padding.Right <= opt.Padding.Left {
		t.Fatalf("padding = %+v", opt.Padding)
	}
}

func TestSanitizeKeepsABottomLegendAtTheBottom(t *testing.T) {
	var o map[string]any
	in := `{"title":{"text":"T","top":15},"legend":{"bottom":0,"left":"center"},"series":[]}`
	if err := json.Unmarshal(Sanitize([]byte(in)), &o); err != nil {
		t.Fatal(err)
	}
	if lg := o["legend"].(map[string]any); lg["top"] != "bottom" {
		t.Fatalf("legend = %v, want top: bottom (go-analyze ignores bottom)", lg)
	}
	if o["title"].(map[string]any)["top"] != 15.0 {
		t.Fatal("a title over a bottom legend keeps its place")
	}
}
