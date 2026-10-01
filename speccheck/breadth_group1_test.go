package speccheck

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// TestGroup1ChartsDrawThroughTheRenderer compiles ECDF, connected scatter and
// bubble requests with the embedded compiler and builds them with the real
// ntcharts renderer: Build must accept the spec and the view must show it.
func TestGroup1ChartsDrawThroughTheRenderer(t *testing.T) {
	ctx := context.Background()
	r, err := compile.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)

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
