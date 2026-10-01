package speccheck

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/ntcharts/v2/canvas"
	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// TestContinuousChartsFillTheWidth compiles line, scatter, sparkline and
// numeric-column heatmap inputs at several sizes with the embedded compiler
// and draws them with the real renderer. flint's layout used to shrink these
// to about 45% of the requested width; the drawn chart must now reach the
// right-hand side of the box it was given.
func TestContinuousChartsFillTheWidth(t *testing.T) {
	ctx := context.Background()
	r, err := compile.New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)

	var rows []string
	for i := 0; i < 40; i++ {
		rows = append(rows, fmt.Sprintf(`{"x":%d,"y":%d}`, i, 30+(i*7)%23+i))
	}
	xy := strings.Join(rows, ",")
	input := func(chartType string) []byte {
		return []byte(fmt.Sprintf(
			`{"data":{"values":[%s]},"chart_spec":{"chartType":%q,"encodings":{"x":{"field":"x"},"y":{"field":"y"}}}}`,
			xy, chartType))
	}
	heat := []byte(`{"data":{"values":[
		{"h":"09","d":"Mon","v":3},{"h":"12","d":"Mon","v":9},{"h":"15","d":"Mon","v":5},
		{"h":"09","d":"Tue","v":7},{"h":"12","d":"Tue","v":2},{"h":"15","d":"Tue","v":8}]},
		"chart_spec":{"chartType":"Heatmap","encodings":{"x":{"field":"h"},"y":{"field":"d"},"color":{"field":"v"}}}}`)

	charts := []struct {
		name  string
		input []byte
	}{
		{"line", input("Line Chart")},
		{"scatter", input("Scatter Plot")},
		{"sparkline", input("Sparkline")},
		{"heatmap", heat},
	}
	for _, c := range charts {
		for _, sz := range []struct{ w, h int }{{40, 12}, {80, 20}, {120, 30}} {
			t.Run(fmt.Sprintf("%s/%dx%d", c.name, sz.w, sz.h), func(t *testing.T) {
				s, _, err := r.Compile(ctx, c.input, envelope.WithBaseSize(sz.w, sz.h))
				if err != nil {
					t.Fatalf("Compile: %v", err)
				}
				if s.Width != sz.w || s.Height != sz.h {
					t.Fatalf("emitted %dx%d, want %dx%d", s.Width, s.Height, sz.w, sz.h)
				}
				model, err := spec.Build(s)
				if err != nil {
					t.Fatalf("Build: %v", err)
				}
				// the drawn chart's widest extent, in cells, must reach near the right edge
				widest := 0
				if hm, ok := model.(*heatmap.Model); ok {
					// heatmap cells are blanks with a background colour, which
					// stripping the view would erase: look at the cells' styles
					for y := 0; y < hm.Height(); y++ {
						for x := 0; x < hm.Width(); x++ {
							bg := hm.Canvas.Cell(canvas.Point{X: x, Y: y}).Style.GetBackground()
							if _, none := bg.(lipgloss.NoColor); !none && bg != nil {
								widest = max(widest, x+1)
							}
						}
					}
				} else {
					for _, line := range strings.Split(ansi.Strip(renderView(t, model)), "\n") {
						widest = max(widest, len([]rune(strings.TrimRight(line, " "))))
					}
				}
				if widest < sz.w-sz.w/10 {
					t.Fatalf("drawn %d cells wide in a %d-wide box", widest, sz.w)
				}
			})
		}
	}
}
