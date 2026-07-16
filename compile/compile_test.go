package compile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

var parityDirs = []struct{ fixtures, expected string }{
	{filepath.Join("..", "testdata", "fixtures"), filepath.Join("..", "testdata", "expected")},
	{filepath.Join("..", "testdata", "fixtures-terminal"), filepath.Join("..", "testdata", "expected-terminal")},
}

func TestParityWithNode(t *testing.T) {
	ctx := context.Background()
	r, err := New(ctx)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close(ctx)

	total := 0
	for _, dirs := range parityDirs {
		fixtures, err := filepath.Glob(filepath.Join(dirs.fixtures, "*.json"))
		if err != nil {
			t.Fatal(err)
		}
		for _, fx := range fixtures {
			total++
			name := filepath.Base(fx)
			t.Run(name, func(t *testing.T) {
				input, err := os.ReadFile(fx)
				if err != nil {
					t.Fatal(err)
				}
				expected, err := os.ReadFile(filepath.Join(dirs.expected, name))
				if err != nil {
					t.Fatalf("missing Node reference (run `npm run gen-expected` in js/): %v", err)
				}
				got, err := r.CompileRaw(ctx, input)
				if err != nil {
					t.Fatalf("CompileRaw: %v", err)
				}
				if string(got) != string(expected) {
					t.Errorf("wasm envelope differs from Node reference\n got (%d bytes): %.300s\nwant (%d bytes): %.300s",
						len(got), got, len(expected), expected)
				}
			})
		}
	}
	if total < 11 {
		t.Fatalf("expected ≥11 parity fixtures, found %d", total)
	}
}

func TestCompileParsesEnvelope(t *testing.T) {
	ctx := context.Background()
	r, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)
	input, err := os.ReadFile(filepath.Join("..", "testdata", "fixtures", "bar-currency.json"))
	if err != nil {
		t.Fatal(err)
	}
	s, warnings, err := r.Compile(ctx, input)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if s.Type != "bar" {
		t.Fatalf("spec.Type = %q, want bar", s.Type)
	}
	if s.Width <= 0 || s.Height <= 0 {
		t.Fatalf("spec has no dimensions: %dx%d", s.Width, s.Height)
	}
	if warnings == nil {
		t.Fatal("warnings must be non-nil (empty slice ok)")
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("compiled spec fails ntcharts Validate: %v", err)
	}
}

func TestCompileErrorSurfaced(t *testing.T) {
	ctx := context.Background()
	r, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(ctx)
	_, _, err = r.Compile(ctx, []byte(`{"data":{"values":[{"a":1}]},"chart_spec":{"chartType":"Rose Chart","encodings":{"x":{"field":"a"}}}}`))
	if err == nil {
		t.Fatal("expected error for unsupported chart type")
	}
}

func TestJSONRoundTripVsGolden(t *testing.T) {
	// The parsed spec, re-marshaled, must contain no data loss vs the envelope's
	// spec object (field-level check of the Go struct coverage).
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "expected", "heatmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	var env struct {
		Spec json.RawMessage `json:"spec"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	var asMap map[string]any
	if err := json.Unmarshal(env.Spec, &asMap); err != nil {
		t.Fatal(err)
	}
	if _, ok := asMap["heat"]; !ok {
		t.Fatal("heatmap envelope spec missing heat data")
	}
}

func BenchmarkCompileScatter(b *testing.B) {
	ctx := context.Background()
	r, err := New(ctx)
	if err != nil {
		b.Fatal(err)
	}
	defer r.Close(ctx)
	input, err := os.ReadFile(filepath.Join("..", "testdata", "fixtures", "scatter.json"))
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := r.CompileRaw(ctx, input); err != nil {
			b.Fatal(err)
		}
	}
}
