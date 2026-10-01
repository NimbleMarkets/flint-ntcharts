package speccheck

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// TestLogScaleDrawsThroughTheRenderer compiles log-scale requests with the
// embedded compiler and builds them with the real ntcharts renderer: the
// emitted scale must be one Build accepts, and the chart must show decade
// labels rather than a squashed linear axis.
func TestLogScaleDrawsThroughTheRenderer(t *testing.T) {
	ctx := context.Background()
	r, err := compile.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)

	rows := func(dated bool) string {
		var vs []string
		for i := 0; i < 10; i++ {
			y := 1.0
			for j := 0; j < i; j++ {
				y *= 3.2
			}
			x := fmt.Sprintf("%d", i+1)
			if dated {
				x = fmt.Sprintf(`"2026-%02d-01"`, i+1)
			}
			vs = append(vs, fmt.Sprintf(`{"x":%s,"y":%.2f}`, x, y))
		}
		return strings.Join(vs, ",")
	}
	chart := func(chartType string, dated bool, props string) []byte {
		return []byte(fmt.Sprintf(
			`{"data":{"values":[%s]},"chart_spec":{"chartType":%q,"chartProperties":%s,"encodings":{"x":{"field":"x"},"y":{"field":"y"}}}}`,
			rows(dated), chartType, props))
	}

	for _, tc := range []struct {
		name  string
		input []byte
	}{
		{"line numeric, log Y", chart("Line Chart", false, `{"logScale_y":true}`)},
		{"line numeric, log X and Y", chart("Line Chart", false, `{"logScale_x":true,"logScale_y":true}`)},
		{"time series, log Y", chart("Line Chart", true, `{"logScale_y":true}`)},
		{"scatter, log X and Y", chart("Scatter Plot", false, `{"logScale_x":true,"logScale_y":true}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, warns, err := r.Compile(ctx, tc.input, envelope.WithBaseSize(60, 16))
			if err != nil {
				t.Fatalf("Compile: %v", err)
			}
			for _, w := range warns {
				if w.Code == "log-scale-unsupported" {
					t.Fatalf("unexpected warning: %s", w.Message)
				}
			}
			if s.YAxis.Scale != spec.ScaleLog {
				t.Fatalf("y_axis.scale = %q, want log", s.YAxis.Scale)
			}
			model, err := spec.Build(s)
			if err != nil {
				t.Fatalf("Build rejected the emitted spec: %v", err)
			}
			view := ansi.Strip(renderView(t, model))
			// ten points spanning ~4.5 decades: whole decades are labelled
			if !strings.Contains(view, "1k") || !strings.Contains(view, "10k") {
				t.Fatalf("no decade labels on the log axis:\n%s", view)
			}
		})
	}
}
