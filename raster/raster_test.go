package raster

import (
	"encoding/json"
	"errors"
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
