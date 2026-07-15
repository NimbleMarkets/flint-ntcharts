package compile

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParityWithNode(t *testing.T) {
	ctx := context.Background()
	r, err := NewRunner(ctx)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	defer r.Close(ctx)

	fixtures, err := filepath.Glob(filepath.Join("..", "testdata", "fixtures", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures found in ../testdata/fixtures")
	}

	for _, fx := range fixtures {
		name := filepath.Base(fx)
		t.Run(name, func(t *testing.T) {
			input, err := os.ReadFile(fx)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := os.ReadFile(filepath.Join("..", "testdata", "expected", name))
			if err != nil {
				t.Fatalf("missing Node reference output (run `npm run gen-expected` in js/): %v", err)
			}
			got, err := r.CompileVegaLite(ctx, input)
			if err != nil {
				t.Fatalf("CompileVegaLite: %v", err)
			}
			if string(got) != string(expected) {
				t.Errorf("wasm output is not byte-identical to Node reference\n got (%d bytes): %.300s\nwant (%d bytes): %.300s",
					len(got), got, len(expected), expected)
			}
		})
	}
}
