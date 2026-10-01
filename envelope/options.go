package envelope

import (
	"encoding/json"
	"fmt"
)

// Option adjusts the ChartAssemblyInput JSON before compilation.
type Option func(doc map[string]any) error

// WithBaseSize sets chart_spec.baseSize (terminal cells). The compiler treats
// this as a hard bound (the backend runs with maxStretch 1).
func WithBaseSize(w, h int) Option {
	return setChartSpecSize("baseSize", w, h)
}

// WithCanvasSize sets chart_spec.canvasSize — the growth ceiling above baseSize.
func WithCanvasSize(w, h int) Option {
	return setChartSpecSize("canvasSize", w, h)
}

// WithRenderer picks the output: "text" (the default, an ntcharts spec) or
// "raster" (flint's ECharts option, for the raster package to draw as an
// image). It sets the input's top-level renderer key.
func WithRenderer(name string) Option {
	return func(doc map[string]any) error {
		if name != "text" && name != "raster" {
			return fmt.Errorf("compile: unknown renderer %q: use \"text\" or \"raster\"", name)
		}
		doc["renderer"] = name
		return nil
	}
}

func setChartSpecSize(key string, w, h int) Option {
	return func(doc map[string]any) error {
		cs, ok := doc["chart_spec"].(map[string]any)
		if !ok {
			return fmt.Errorf("compile: input has no chart_spec object")
		}
		cs[key] = map[string]any{"width": w, "height": h}
		return nil
	}
}

// Apply returns input unchanged when no options are given (preserving
// byte-parity paths); otherwise parses, applies each option, re-serializes.
func Apply(input []byte, opts []Option) ([]byte, error) {
	if len(opts) == 0 {
		return input, nil
	}
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		return nil, fmt.Errorf("compile: input is not valid JSON: %w", err)
	}
	for _, opt := range opts {
		if err := opt(doc); err != nil {
			return nil, err
		}
	}
	return json.Marshal(doc)
}
