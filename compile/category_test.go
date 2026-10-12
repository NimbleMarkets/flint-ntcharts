package compile

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/charmbracelet/x/ansi"
)

func TestCompileCategoricalCharts(t *testing.T) {
	ctx := context.Background()
	r, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)
	for _, name := range []string{"line-categorical", "scatter-categorical", "slope", "strip", "strip-dense"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile("../testdata/fixtures-terminal/" + name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			for _, size := range [][2]int{{80, 24}, {44, 15}, {20, 8}} {
				s, _, err := r.Compile(ctx, raw, WithBaseSize(size[0], size[1]))
				if err != nil {
					t.Fatal(err)
				}
				if s.XAxis.Type != spec.XAxisCategory || len(s.XAxis.Labels) < 2 {
					t.Fatalf("lost categories: %+v", s.XAxis)
				}
				model, err := spec.Build(s)
				if err != nil {
					t.Fatal(err)
				}
				view := model.(interface{ View() string }).View()
				if strings.TrimSpace(view) == "" {
					t.Fatal("empty categorical plot")
				}
				lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
				if len(lines) > size[1] {
					t.Fatalf("height %d exceeds %d", len(lines), size[1])
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > size[0] {
						t.Fatal("plot exceeds requested width")
					}
				}
				if size[0] >= 44 {
					for _, label := range s.XAxis.Labels {
						if !strings.Contains(ansi.Strip(view), label) {
							t.Fatalf("missing label %q", label)
						}
					}
				}
			}
		})
	}
}
