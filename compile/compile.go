// Package compile runs the flint→ntcharts compiler, embedded as a Javy-built
// WASI module, via wazero. Pure Go: no cgo, no Node at runtime. Output is the
// ntcharts-spec envelope {spec, warnings, size}.
package compile

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"

	"github.com/NimbleMarkets/flint-ntcharts/envelope"
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
	rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
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
// compiler and returns the raw ntcharts-spec envelope JSON it emits,
// verbatim, without inspecting its top-level key. On a compile-time failure
// the returned bytes are still a well-formed envelope (an `{"error": ...}`
// object rather than `{"spec": ...}`) and err is nil; callers that want that
// distinction surfaced as a Go error should use Compile instead, or
// discriminate the raw bytes themselves. err is non-nil only for the wasm
// process/TRAP failure layer (module instantiation failure, non-zero/non-WASI
// exit, or context cancellation), not for compile-time failures.
func (r *Runner) CompileRaw(ctx context.Context, input []byte, opts ...Option) ([]byte, error) {
	input, err := envelope.Apply(input, opts)
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
//
// Deprecated: alias of envelope.Warning; use the envelope package directly in
// new code.
type Warning = envelope.Warning

// Compile runs the embedded compiler and parses the envelope. Warnings is
// never nil. The returned spec has already been shaped by flint; callers
// typically pass it straight to ntcharts spec.Build.
//
// Compile errors come in two layers. A compile-time failure (e.g. an
// unsupported chart type) still produces a clean, valid envelope on stdout —
// this is reported here as an error built from the envelope's `error.message`
// field. A wasm TRAP (an engine-level failure such as OOM) instead surfaces
// as a process error from CompileRaw, with stderr attached.
//
// ctx is honored for cancellation: the underlying wazero runtime is created
// with WithCloseOnContextDone(true), so an already-canceled or later-canceled
// ctx aborts the compile and Compile returns a non-nil error rather than a
// successful result.
func (r *Runner) Compile(ctx context.Context, input []byte, opts ...Option) (spec.Spec, []Warning, error) {
	raw, err := r.CompileRaw(ctx, input, opts...)
	if err != nil {
		return spec.Spec{}, nil, err
	}
	return envelope.Parse(raw)
}
