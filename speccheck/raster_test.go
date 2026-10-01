package speccheck

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/flint-ntcharts/raster"
)

// TestRasterRendererEndToEnd compiles with the real embedded compiler, asking
// for the raster renderer, and draws the result with the raster package: the
// whole route from a flint spec to an image.
func TestRasterRendererEndToEnd(t *testing.T) {
	ctx := context.Background()
	r, err := compile.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)

	grouped := `{"data":{"values":[
		{"c":"Laptop","n":3,"g":"USA"},{"c":"Phone","n":5,"g":"USA"},{"c":"Laptop","n":2,"g":"China"},{"c":"Phone","n":4,"g":"China"}]},
		"chart_spec":{"chartType":"Grouped Bar Chart","encodings":{"x":{"field":"c"},"y":{"field":"n"},"group":{"field":"g"}}}}`

	t.Run("a grouped bar chart, which text cannot draw, draws as an image", func(t *testing.T) {
		res, err := r.CompileResult(ctx, []byte(grouped), envelope.WithRenderer("raster"), envelope.WithBaseSize(80, 24))
		if err != nil {
			t.Fatalf("CompileResult: %v", err)
		}
		if res.Spec != nil || res.ECharts == nil {
			t.Fatalf("want an echarts result, got %+v", res)
		}
		if res.Size.Width != 80 || res.Size.Height != 24 {
			t.Fatalf("size = %+v", res.Size)
		}
		img, err := raster.Render(res.ECharts, 640, 400)
		if err != nil {
			t.Fatalf("Render: %v", err)
		}
		if img.Bounds().Dx() != 640 {
			t.Fatalf("bounds = %v", img.Bounds())
		}
	})

	t.Run("the same chart without a renderer is an error that points at raster", func(t *testing.T) {
		_, _, err := r.Compile(ctx, []byte(grouped))
		if err == nil || !strings.Contains(err.Error(), `"renderer": "raster"`) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("Compile rejects a raster envelope instead of returning an empty spec", func(t *testing.T) {
		_, _, err := r.Compile(ctx, []byte(grouped), envelope.WithRenderer("raster"))
		if err == nil || !strings.Contains(err.Error(), "ParseResult") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("a chart go-analyze draws blank is reported as blank", func(t *testing.T) {
		waterfall := `{"data":{"values":[{"step":"Revenue","amt":100},{"step":"Costs","amt":-40},{"step":"Tax","amt":-10},{"step":"Net","amt":50}]},
			"chart_spec":{"chartType":"Waterfall Chart","encodings":{"x":{"field":"step"},"y":{"field":"amt"}}}}`
		res, err := r.CompileResult(ctx, []byte(waterfall), envelope.WithRenderer("raster"))
		if err != nil {
			t.Fatalf("CompileResult: %v", err)
		}
		if _, err := raster.Render(res.ECharts, 640, 400); !errors.Is(err, raster.ErrBlank) {
			t.Fatalf("Render err = %v, want ErrBlank", err)
		}
	})

	t.Run("a text-supported chart still compiles to a spec by default", func(t *testing.T) {
		bar := `{"data":{"values":[{"m":"Jan","r":1},{"m":"Feb","r":2}]},"chart_spec":{"chartType":"Bar Chart","encodings":{"x":{"field":"m"},"y":{"field":"r"}}}}`
		res, err := r.CompileResult(ctx, []byte(bar))
		if err != nil || res.Spec == nil || res.ECharts != nil {
			t.Fatalf("res = %+v, err = %v", res, err)
		}
	})
}
