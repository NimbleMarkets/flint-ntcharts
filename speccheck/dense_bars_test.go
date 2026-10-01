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

// TestDenseBarsDrawBars runs bar charts with far more categories than the
// terminal can show through the embedded compiler and the real renderer, and
// requires actual bars on screen. The ntcharts bar model draws nothing but
// the axis once its bar width reaches zero, and a "non-empty view" check
// cannot tell that apart from a chart — which is how the compiler used to
// emit 60 categories into a 20-cell chart unnoticed.
func TestDenseBarsDrawBars(t *testing.T) {
	ctx := context.Background()
	r, err := compile.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)

	barInput := func(n int, horizontal bool) []byte {
		var rows []string
		for i := 0; i < n; i++ {
			rows = append(rows, fmt.Sprintf(`{"product":"item-%03d","revenue":%d}`, i, 10+i%17))
		}
		x, y := "product", "revenue"
		if horizontal {
			x, y = y, x
		}
		return []byte(fmt.Sprintf(
			`{"data":{"values":[%s]},"chart_spec":{"chartType":"Bar Chart","encodings":{"x":{"field":%q},"y":{"field":%q}}}}`,
			strings.Join(rows, ","), x, y))
	}

	sizes := []struct{ w, h int }{{12, 6}, {20, 8}, {40, 12}, {64, 20}}
	for _, horizontal := range []bool{false, true} {
		for _, n := range []int{5, 19, 40, 100, 300} {
			for _, sz := range sizes {
				name := fmt.Sprintf("n=%d/%dx%d/horizontal=%v", n, sz.w, sz.h, horizontal)
				t.Run(name, func(t *testing.T) {
					s, _, err := r.Compile(ctx, barInput(n, horizontal), envelope.WithBaseSize(sz.w, sz.h))
					if err != nil {
						t.Fatalf("Compile: %v", err)
					}
					model, err := spec.Build(s)
					if err != nil {
						t.Fatalf("Build: %v", err)
					}
					view := ansi.Strip(renderView(t, model))
					if !hasBlockElement(view) {
						t.Fatalf("%d categories emitted for a %dx%d chart, but no bar was drawn:\n%s",
							len(s.XAxis.Labels), s.Width, s.Height, view)
					}
				})
			}
		}
	}
}

// hasBlockElement reports whether s contains a Unicode block element
// (U+2580–U+259F), the glyphs the bar model draws bars with.
func hasBlockElement(s string) bool {
	for _, r := range s {
		if r >= 0x2580 && r <= 0x259F {
			return true
		}
	}
	return false
}
