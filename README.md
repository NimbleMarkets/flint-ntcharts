# flint-ntcharts

A spike proving that [flint-chart](https://github.com/NimbleMarkets/flint-chart) can compile
Vega-Lite specs inside a [Javy](https://github.com/bytecodealliance/javy)/QuickJS WebAssembly
module, run from Go via [wazero](https://github.com/tetratelabs/wazero) — no Node.js runtime
required at execution time. This is Phase 1 (feasibility) of the flint → ntcharts backend
project: proving the wasm approach is viable before committing to it as the production
compile path. See [SPIKE-RESULTS.md](./SPIKE-RESULTS.md) for the full go/no-go verdict,
benchmark numbers, and known caveats.

## Quickstart

The compiled `flint.wasm` module is committed to the repo, so a fresh clone can run the Go
test suite (parity tests + benchmark) with no build step:

```sh
go test ./...
```

### Rebuilding the wasm module

If you change `js/src/*.js` (or want to reproduce the artifact yourself):

```sh
cd js && npm install
make wasm   # downloads bin/javy via `gh` on first run (macOS arm64)
```

### Regenerating reference fixtures

The Node-side reference outputs used by the Go parity test live under `testdata/expected/`.
To regenerate them from `flint-chart` directly:

```sh
cd js && npm run gen-expected
```

## Requirements

- Go >= 1.25 (required by `wazero`)
- Node >= 20
- `gh` CLI on your `PATH` if you need `make wasm` to fetch `bin/javy` (macOS arm64 only; other
  platforms should set `JAVY` to a pre-installed binary)
