// Package speccheck cross-validates the TypeScript backend's emitted
// ntcharts-spec JSON against the real Go renderer: every golden must pass
// Validate() and Build() and produce a non-empty terminal view.
package speccheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

func TestGoldensBuildWithNtcharts(t *testing.T) {
	// Glob both the pixel-scale (testdata/ntspec-golden) and terminal-scale
	// (testdata/ntspec-golden-terminal) golden directories: the terminal
	// suite (Fix 2/3) exercises the maxStretch:1 default base-size path and
	// the quantitative-x line/wavelinechart build, on top of the original
	// pixel-scale fixtures.
	var goldens []string
	for _, dir := range []string{"ntspec-golden", "ntspec-golden-terminal"} {
		matches, err := filepath.Glob(filepath.Join("..", "testdata", dir, "*.json"))
		if err != nil {
			t.Fatal(err)
		}
		goldens = append(goldens, matches...)
	}
	if len(goldens) < 13 {
		t.Fatalf("expected ≥13 goldens across ntspec-golden + ntspec-golden-terminal, found %d (run the vitest suite first)", len(goldens))
	}
	for _, path := range goldens {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var s spec.Spec
			if err := json.Unmarshal(raw, &s); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if err := s.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			model, err := spec.Build(s)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			view := renderView(t, model)
			if strings.TrimSpace(view) == "" {
				t.Fatal("rendered view is empty")
			}
		})
	}
}

// renderView calls the model's View() string method via an interface
// assertion (the model's concrete type satisfies an ad-hoc `viewer`
// interface), not a type switch on the concrete types spec.Build can return.
func renderView(t *testing.T, model any) string {
	t.Helper()
	type viewer interface{ View() string }
	if v, ok := model.(viewer); ok {
		return v.View()
	}
	t.Fatalf("model %T has no View() string", model)
	return ""
}
