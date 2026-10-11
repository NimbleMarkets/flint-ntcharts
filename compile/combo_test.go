package compile

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

func TestCompileCombo(t *testing.T) {
	ctx := context.Background()
	r, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)
	for _, fixture := range []string{"combo", "combo-sparse", "combo-bars"} {
		t.Run(fixture, func(t *testing.T) {
			raw, err := os.ReadFile("../testdata/fixtures-terminal/" + fixture + ".json")
			if err != nil {
				t.Fatal(err)
			}
			for _, size := range [][2]int{{80, 24}, {44, 15}, {100, 30}} {
				s, warnings, err := r.Compile(ctx, raw, WithBaseSize(size[0], size[1]))
				if err != nil || len(warnings) != 0 {
					t.Fatalf("compile: %v, warnings %v", err, warnings)
				}
				if fixture != "combo-bars" {
					c, err := spec.ResolveCategoricalCombo(s)
					if err != nil {
						t.Fatal(err)
					}
					if len(c.Labels) != 6 || c.Min > 0 || c.Max < 0 {
						t.Fatalf("invalid combo: %+v", c)
					}
					if fixture == "combo-sparse" && (c.Bars[2] != nil || c.Line[1] != nil || *c.Bars[0] != -25) {
						t.Fatalf("lost sparse/negative data: %+v", c)
					}
				} else if s.Y2Axis != nil || len(s.Data.Series) != 1 {
					t.Fatal("expected bars-only fallback")
				}
				m, err := spec.Build(s)
				if err != nil {
					t.Fatal(err)
				}
				if strings.TrimSpace(m.(interface{ View() string }).View()) == "" {
					t.Fatal("empty chart")
				}
			}
		})
	}
}
