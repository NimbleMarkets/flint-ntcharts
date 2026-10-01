package speccheck

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// TestBreadthChartsDrawThroughTheRenderer compiles ECDF, connected scatter, bubble,
// histogram, area and lollipop requests with the embedded compiler and builds them with the real
// ntcharts renderer: Build must accept the spec and the view must show it.
func TestBreadthChartsDrawThroughTheRenderer(t *testing.T) {
	ctx := context.Background()
	r, err := compile.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)

	calendar := func(days int) string {
		var vs []string
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < days; i++ {
			vs = append(vs, fmt.Sprintf(`{"d":%q,"n":%d}`, start.AddDate(0, 0, i).Format("2006-01-02"), (i*7)%13+i%3))
		}
		return `{"data":{"values":[` + strings.Join(vs, ",") + `]},"chart_spec":{"chartType":"Calendar Heatmap","encodings":{"x":{"field":"d"},"color":{"field":"n"}}}}`
	}

	for _, tc := range []struct {
		name  string
		input string
		want  []string // substrings the stripped view must contain
	}{
		{
			"ecdf",
			`{"data":{"values":[{"v":3},{"v":1},{"v":2},{"v":2},{"v":9},{"v":4},{"v":7},{"v":5}]},
			  "chart_spec":{"chartType":"ECDF Plot","encodings":{"x":{"field":"v"}}}}`,
			[]string{"100%", "0%"},
		},
		{
			"connected scatter",
			`{"data":{"values":[{"x":1,"y":1,"t":0},{"x":4,"y":3,"t":1},{"x":2,"y":5,"t":2},{"x":6,"y":2,"t":3}]},
			  "chart_spec":{"chartType":"Connected Scatter Plot","encodings":{"x":{"field":"x"},"y":{"field":"y"},"order":{"field":"t"}}}}`,
			nil,
		},
		{
			"histogram",
			`{"data":{"values":[{"v":1},{"v":2},{"v":2},{"v":3},{"v":3},{"v":3},{"v":4},{"v":4},{"v":5},{"v":9}]},
			  "chart_spec":{"chartType":"Histogram","chartProperties":{"binCount":4},"encodings":{"x":{"field":"v"}}}}`,
			[]string{"1–3", "7–9"},
		},
		{
			"area",
			`{"data":{"values":[{"x":1,"y":2},{"x":2,"y":5},{"x":3,"y":3},{"x":4,"y":6}]},
			  "chart_spec":{"chartType":"Area Chart","encodings":{"x":{"field":"x"},"y":{"field":"y"}}}}`,
			nil,
		},
		{
			"lollipop",
			`{"data":{"values":[{"c":"a","n":3},{"c":"b","n":5},{"c":"c","n":2}]},
			  "chart_spec":{"chartType":"Lollipop Chart","encodings":{"x":{"field":"c"},"y":{"field":"n"}}}}`,
			[]string{"a", "b", "c"},
		},
		{"calendar, 12 weeks", calendar(84), []string{"Mon", "Sun", "Jan", "Feb", "Mar"}},
		{"calendar, a year", calendar(365), []string{"Mon", "Sun", "Jan"}},
		{
			"bubble",
			`{"data":{"values":[{"x":1,"y":1,"s":5},{"x":4,"y":3,"s":50},{"x":2,"y":5,"s":20}]},
			  "chart_spec":{"chartType":"Bubble Chart","encodings":{"x":{"field":"x"},"y":{"field":"y"},"size":{"field":"s"}}}}`,
			nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, err := r.Compile(ctx, []byte(tc.input), envelope.WithBaseSize(60, 16))
			if err != nil {
				t.Fatalf("Compile: %v", err)
			}
			model, err := spec.Build(s)
			if err != nil {
				t.Fatalf("Build rejected the emitted spec: %v", err)
			}
			view := ansi.Strip(renderView(t, model))
			if strings.TrimSpace(view) == "" {
				t.Fatal("empty view")
			}
			for _, w := range tc.want {
				if !strings.Contains(view, w) {
					t.Fatalf("view lacks %q:\n%s", w, view)
				}
			}
			t.Log("\n" + view)
		})
	}
}
