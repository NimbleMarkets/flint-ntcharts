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
	// Error is set instead of Spec/Warnings/Size when the wasm/Node compiler
	// hits a compile-time failure (e.g. an unsupported chart type): it still
	// writes valid JSON to stdout and exits 0, so this is discriminated by
	// the top-level key present in the raw JSON, not by process exit status.
	// A wasm TRAP (engine-level failure, e.g. OOM) is a separate failure
	// layer: it does not produce this envelope at all and instead surfaces
	// as the CompileRaw error path above (stderr attached).
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

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
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		// %.200s is a safe truncation of the raw bytes for the error message:
		// %s on a []byte prints it like a string (Go's fmt does not require
		// valid UTF-8 to do this), and the precision caps the byte count, so
		// this can never panic or produce invalid output regardless of what
		// the wasm module wrote to stdout.
		return spec.Spec{}, nil, fmt.Errorf("compile: bad envelope: %w (raw: %.200s)", err, raw)
	}
	if env.Error != nil {
		return spec.Spec{}, nil, fmt.Errorf("flint compile: %s", env.Error.Message)
	}
	if env.Warnings == nil {
		env.Warnings = []Warning{}
	}
	return env.Spec, env.Warnings, nil
}
