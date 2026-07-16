package compile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
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
	if !strings.Contains(err.Error(), "Unknown chart type") {
		t.Fatalf("error = %q, want it to contain %q", err.Error(), "Unknown chart type")
	}
}

func TestCompileCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	r, err := New(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close(context.Background())

	input, err := os.ReadFile(filepath.Join("..", "testdata", "fixtures", "scatter.json"))
	if err != nil {
		t.Fatal(err)
	}

	cancel() // cancel before the compile even starts

	if _, _, err := r.Compile(ctx, input); err == nil {
		t.Fatal("expected an error compiling with an already-canceled context, got nil")
	}
}

// keyPathsSubset walks every key path present in want and asserts the same
// path is present (non-nil container, key exists) in got. Values are allowed
// to differ in formatting (numbers vs strings, float precision, etc) -- this
// only catches a Go struct silently dropping a field the frozen envelope
// contract still emits.
func keyPathsSubset(t *testing.T, path string, want, got any) {
	t.Helper()
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			t.Errorf("%s: want object, re-marshaled got %T", path, got)
			return
		}
		for k, wv := range w {
			gv, ok := g[k]
			if !ok {
				t.Errorf("%s.%s: key dropped by re-marshal", path, k)
				continue
			}
			keyPathsSubset(t, path+"."+k, wv, gv)
		}
	case []any:
		g, ok := got.([]any)
		if !ok {
			t.Errorf("%s: want array, re-marshaled got %T", path, got)
			return
		}
		if len(g) != len(w) {
			t.Errorf("%s: array length changed: want %d, got %d", path, len(w), len(g))
			return
		}
		for i, wv := range w {
			keyPathsSubset(t, fmt.Sprintf("%s[%d]", path, i), wv, g[i])
		}
	}
	// scalars: presence at this path was already confirmed by the caller.
}

func TestSpecFieldDrift(t *testing.T) {
	// For every committed reference envelope, decode its `spec` object into
	// ntcharts spec.Spec, re-marshal that Go value, and confirm every key
	// path present in the original spec JSON survives in the re-marshaled
	// JSON. This catches the Go struct silently dropping/renaming a field
	// relative to the frozen wasm/Node envelope contract -- TestParityWithNode
	// only checks wasm-vs-Node byte equality, not Go-struct coverage of it.
	dirs := []string{
		filepath.Join("..", "testdata", "expected"),
		filepath.Join("..", "testdata", "expected-terminal"),
	}
	total := 0
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			t.Fatal(err)
		}
		for _, fx := range files {
			total++
			name := filepath.Join(filepath.Base(dir), filepath.Base(fx))
			t.Run(name, func(t *testing.T) {
				raw, err := os.ReadFile(fx)
				if err != nil {
					t.Fatal(err)
				}
				var env struct {
					Spec spec.Spec `json:"spec"`
				}
				if err := json.Unmarshal(raw, &env); err != nil {
					t.Fatalf("decode envelope: %v", err)
				}

				var envRaw struct {
					Spec json.RawMessage `json:"spec"`
				}
				if err := json.Unmarshal(raw, &envRaw); err != nil {
					t.Fatalf("decode envelope raw: %v", err)
				}
				var want any
				if err := json.Unmarshal(envRaw.Spec, &want); err != nil {
					t.Fatalf("decode original spec JSON: %v", err)
				}

				remarshaled, err := json.Marshal(env.Spec)
				if err != nil {
					t.Fatalf("re-marshal spec.Spec: %v", err)
				}
				var got any
				if err := json.Unmarshal(remarshaled, &got); err != nil {
					t.Fatalf("decode re-marshaled spec JSON: %v", err)
				}

				keyPathsSubset(t, "spec", want, got)
			})
		}
	}
	if total < 11 {
		t.Fatalf("expected >=11 reference envelopes, found %d", total)
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
