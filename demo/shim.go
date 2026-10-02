//go:build js && wasm

package main

import (
	"context"

	"github.com/NimbleMarkets/booba-shim/flintchart"
	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// shimCompiler compiles through the booba-shim flintchart bridge: the flint
// compiler runs as the vendored JavaScript bundle on the page.
type shimCompiler struct{}

func (shimCompiler) Compile(ctx context.Context, input []byte, opts ...envelope.Option) (spec.Spec, []envelope.Warning, error) {
	input, err := envelope.Apply(input, opts)
	if err != nil {
		return spec.Spec{}, nil, err
	}
	raw, err := flintchart.Compile(ctx, input)
	if err != nil {
		return spec.Spec{}, nil, err
	}
	return envelope.Parse(raw)
}

// CompileResult is Compile for either renderer, so the viewer can draw
// "renderer": "raster" documents as images. It needs a shim bundle built from
// flint-ntcharts v0.3.0 or later; an older one answers with an error, and the
// viewer falls back to the text chart.
func (shimCompiler) CompileResult(ctx context.Context, input []byte, opts ...envelope.Option) (envelope.Result, error) {
	input, err := envelope.Apply(input, opts)
	if err != nil {
		return envelope.Result{}, err
	}
	raw, err := flintchart.Compile(ctx, input)
	if err != nil {
		return envelope.Result{}, err
	}
	return envelope.ParseResult(raw)
}
