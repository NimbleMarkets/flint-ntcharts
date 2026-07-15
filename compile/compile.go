// Package compile runs the flint-chart compiler, embedded as a Javy-built
// WASI module, via wazero. Pure Go: no cgo, no Node at runtime.
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
)

//go:embed flint.wasm
var flintWasm []byte

// Runner holds a compiled flint.wasm module ready for repeated execution.
// Each CompileVegaLite call instantiates a fresh module instance, so a
// Runner is safe for concurrent use.
type Runner struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
}

func NewRunner(ctx context.Context) (*Runner, error) {
	rt := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	compiled, err := rt.CompileModule(ctx, flintWasm)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("compile flint.wasm: %w", err)
	}
	return &Runner{runtime: rt, compiled: compiled}, nil
}

func (r *Runner) Close(ctx context.Context) error {
	return r.runtime.Close(ctx)
}

// CompileVegaLite feeds a flint ChartAssemblyInput JSON document to the
// embedded compiler and returns the Vega-Lite spec JSON it emits.
func (r *Runner) CompileVegaLite(ctx context.Context, input []byte) ([]byte, error) {
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
