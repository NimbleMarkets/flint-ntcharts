package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
	"github.com/charmbracelet/x/ansi"
)

// v2.7 silently ignores the new JSON fields. Keep the published dependency
// usable while exercising unreleased dual axes whenever the local workspace
// selects an ntcharts revision that supports them.
func TestRenderDualAxes(t *testing.T) {
	if _, ok := reflect.TypeFor[spec.Spec]().FieldByName("Y2Axis"); !ok {
		t.Skip("dual axes require local ntcharts: go work init . ../ntcharts")
	}
	for _, kind := range []string{"line", "scatter", "timeseries"} {
		t.Run(kind, func(t *testing.T) {
			raw, err := os.ReadFile("../testdata/dual-axis/" + kind + ".json")
			if err != nil {
				t.Fatal(err)
			}
			for _, envelope := range []bool{false, true} {
				for _, size := range [][2]int{{80, 24}, {44, 15}, {100, 30}} {
					t.Run(fmt.Sprintf("envelope=%v/%dx%d", envelope, size[0], size[1]), func(t *testing.T) {
						doc := raw
						if envelope {
							doc = []byte(`{"spec":` + string(raw) + `,"warnings":[]}`)
						}
						// A nil compiler proves these inputs use the native spec path.
						view, warnings, err := Render(nil, doc, size[0], size[1])
						if err != nil || len(warnings) != 0 {
							t.Fatalf("render: %v, warnings: %v", err, warnings)
						}
						plain := ansi.Strip(view)
						if !strings.Contains(plain, "100%") || !strings.Contains(plain, "30") {
							t.Fatalf("missing independent left/right tick labels:\n%s", plain)
						}
						lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
						if len(lines) > size[1] {
							t.Fatalf("height %d exceeds %d", len(lines), size[1])
						}
						for _, line := range lines {
							if width := ansi.StringWidth(line); width > size[0] {
								t.Fatalf("width %d exceeds %d", width, size[0])
							}
						}
					})
				}
			}
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			delete(doc, "y2_axis")
			invalid, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := Render(nil, invalid, 80, 24); err == nil || !strings.Contains(err.Error(), "requires y2_axis") {
				t.Fatalf("right-hand series without an axis: %v", err)
			}
		})
	}
}
