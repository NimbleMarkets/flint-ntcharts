// Package compile runs the flint→ntcharts compiler, embedded as a Javy-built
// WASI module, via wazero. Pure Go: no cgo, no Node at runtime. Output is the
// ntcharts-spec envelope {spec, warnings, size}.
package compile

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

//go:embed flint.wasm
var flintWasm []byte

// Runner holds a compiled flint.wasm module ready for repeated execution.
// Each CompileRaw call instantiates a fresh module instance, so a Runner is
// safe for concurrent use.
type Runner struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
}

// New compiles the embedded flint.wasm module and returns a Runner ready for
// repeated Compile/CompileRaw calls.
func New(ctx context.Context) (*Runner, error) {
	rt := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	compiled, err := rt.CompileModule(ctx, flintWasm)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("compile flint.wasm: %w", err)
	}
	return &Runner{runtime: rt, compiled: compiled}, nil
}

// NewRunner constructs a Runner.
//
// Deprecated: use New.
func NewRunner(ctx context.Context) (*Runner, error) {
	return New(ctx)
}

func (r *Runner) Close(ctx context.Context) error {
	return r.runtime.Close(ctx)
}

// CompileRaw feeds a flint ChartAssemblyInput JSON document to the embedded
// compiler and returns the raw ntcharts-spec envelope JSON it emits.
func (r *Runner) CompileRaw(ctx context.Context, input []byte, opts ...Option) ([]byte, error) {
	input, err := applyOptions(input, opts)
	if err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	cfg := wazero.NewModuleConfig().
		WithStdin(bytes.NewReader(input)).
		WithStdout(&stdout).
		WithStderr(&stderr).
		WithName("") // anonymous instance: allows concurrent runs

	mod, err := r.runtime.InstantiateModule(ctx, r.compiled, cfg)
	if mod != nil {
		defer mod.Close(ctx)
	}
	if err != nil {
		var exitErr *sys.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 0 {
			// clean WASI exit
		} else {
			return nil, fmt.Errorf("flint wasm: %w (stderr: %s)", err, stderr.String())
		}
	}
	if stdout.Len() == 0 {
		return nil, fmt.Errorf("flint wasm produced no output (stderr: %s)", stderr.String())
	}
	return stdout.Bytes(), nil
}

// Warning mirrors flint's ChartWarning wire shape.
type Warning struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Channel  string `json:"channel,omitempty"`
	Field    string `json:"field,omitempty"`
}

type envelope struct {
	Spec     spec.Spec `json:"spec"`
	Warnings []Warning `json:"warnings"`
	Size     struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"size"`
}

// Compile runs the embedded compiler and parses the envelope. Warnings is
// never nil. The returned spec has already been shaped by flint; callers
// typically pass it straight to ntcharts spec.Build.
func (r *Runner) Compile(ctx context.Context, input []byte, opts ...Option) (spec.Spec, []Warning, error) {
	raw, err := r.CompileRaw(ctx, input, opts...)
	if err != nil {
		return spec.Spec{}, nil, err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return spec.Spec{}, nil, fmt.Errorf("compile: bad envelope: %w (raw: %.200s)", err, raw)
	}
	if env.Warnings == nil {
		env.Warnings = []Warning{}
	}
	return env.Spec, env.Warnings, nil
}
