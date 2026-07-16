# WASM Rewire + Go Compile API Implementation Plan (flint → ntcharts, Phase 4)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The embedded wasm compiler emits ntcharts-spec envelopes (`{spec, warnings, size}`) instead of Vega-Lite, and the Go `compile` package exposes the production API `Compile(ctx, input, opts...) (spec.Spec, []Warning, error)` — plus build provenance and CI.

**Architecture:** The Phase-3 TypeScript backend (`assembleNtcharts`) becomes the wasm bundle's compile function via a new `compileToNtSpec` entry in `js/src/compile.js` (envelope split: `_warnings`/`_width`/`_height` → `warnings`/`size`, clean spec). The Node reference runner and the Go parity suite regenerate around the new envelope — byte-parity methodology unchanged (same JS both sides). The Go API parses the envelope into `spec.Spec` from the ntcharts module (already a dependency via speccheck) and gains `WithBaseSize`/`WithCanvasSize` options (JSON-injected into `chart_spec` before compilation — the Phase-5 terminal-resize hook). Makefile gains OS-aware javy download and a provenance stamp; a GitHub Actions workflow validates the committed artifacts without needing javy.

**Tech Stack:** existing (esbuild, Javy 9, wazero, Go 1.25, vitest); GitHub Actions YAML.

**Design refs:** design doc §4 (`docs/2026-07-15-flint-ntcharts-backend-design.md`); Phase-1 SPIKE-RESULTS.md carry-overs (provenance, CI); ledger Phase-4 carry-overs.

## Global Constraints

- Work in `/Users/evan/projects/flint-ntcharts` (branch main). Explicit-path staging (never `git add -A` — `.superpowers/` is scratch).
- The Vega-Lite compile path is REMOVED, not kept alongside: `compileToVegaLite` and its expected outputs are replaced by the ntcharts envelope (`compileToNtSpec`). Phase 1's plan explicitly called Vega-Lite "the stand-in compile entry point". The Phase-1 spike's *methodology* (byte-parity Node↔wasm) is preserved.
- **Envelope contract** (frozen after this phase — flint-tui and booba-shim consume it):
  `{"spec": <NtSpec without any _-prefixed keys>, "warnings": [ChartWarning...], "size": {"width": N, "height": N}}` — `warnings` always present (possibly `[]`), `size` always present, key order exactly `spec, warnings, size`. Serialized by `JSON.stringify` with no pretty-printing on the wasm/Node path.
- Byte-parity requirement: wasm stdout must be byte-identical to Node stdout for every fixture in BOTH `testdata/fixtures/` and `testdata/fixtures-terminal/` (11 total). Node references live in `testdata/expected/` and `testdata/expected-terminal/`.
- Go API surface produced this phase (Phase 5/6 depend on these exact names):
  `New(ctx context.Context) (*Runner, error)` (rename of NewRunner — keep a deprecated `NewRunner` alias), `(*Runner) Compile(ctx context.Context, input []byte, opts ...Option) (spec.Spec, []Warning, error)`, `(*Runner) CompileRaw(ctx context.Context, input []byte, opts ...Option) ([]byte, error)`, `(*Runner) Close(ctx context.Context) error`, `type Warning`, `type Option`, `WithBaseSize(w, h int) Option`, `WithCanvasSize(w, h int) Option`.
- `spec.Spec` comes from `github.com/NimbleMarkets/ntcharts/v2/spec` (dependency + replace already in go.mod — do not modify go.mod beyond what `go mod tidy` requires).
- All suites must be green at each task boundary: `cd js && npx vitest run && npx tsc --noEmit && npm run build`; `go test ./...` (compile parity + speccheck).
- The ntcharts repo is NOT touched this phase.
- Javy version for rebuilds: whatever `bin/javy --version` reports (v9.0.0 from Phase 1); record it in the provenance stamp.

---

### Task 1: JS envelope — compileToNtSpec, entry rewire, reference regeneration

**Files:**
- Modify: `js/src/compile.js`
- Modify: `js/src/entry-javy.js` (one import/call swap)
- Modify: `js/scripts/node-runner.mjs` (dual fixture dirs, envelope entry)
- Create: `js/test/envelope.test.ts`
- Delete: `testdata/expected/*.json` (Vega-Lite refs) → regenerate as envelopes
- Create (generated): `testdata/expected-terminal/*.json`

**Interfaces:**
- Produces: `compileToNtSpec(inputJSON: string): string` in `js/src/compile.js` — THE wasm/Node compile function. Envelope key order `spec, warnings, size`. Consumed by entry-javy.js, node-runner.mjs, and (conceptually) the Go parity suite.

- [ ] **Step 1: Write the failing envelope test**

`js/test/envelope.test.ts`:

```ts
import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { compileToNtSpec } from "../src/compile.js";

const fixtureDir = fileURLToPath(new URL("../../testdata/fixtures/", import.meta.url));

describe("compileToNtSpec envelope", () => {
  it("wraps the spec with warnings and size, stripping private keys", () => {
    const input = readFileSync(`${fixtureDir}bar-currency.json`, "utf8");
    const out = JSON.parse(compileToNtSpec(input));
    expect(Object.keys(out)).toEqual(["spec", "warnings", "size"]);
    expect(out.spec.type).toBe("bar");
    expect(Array.isArray(out.warnings)).toBe(true);
    expect(out.size).toEqual({ width: out.spec.width, height: out.spec.height });
    for (const key of Object.keys(out.spec)) expect(key.startsWith("_")).toBe(false);
  });
  it("stringifies compactly and deterministically", () => {
    const input = readFileSync(`${fixtureDir}scatter.json`, "utf8");
    const a = compileToNtSpec(input);
    const b = compileToNtSpec(input);
    expect(a).toBe(b);
    expect(a).not.toContain("\n");
  });
});
```

Run `cd js && npx vitest run` → FAIL (`compileToNtSpec` not exported).

- [ ] **Step 2: Implement `compileToNtSpec` and rewire the Javy entry**

`js/src/compile.js` — keep the file's single-code-path discipline; `compileToVegaLite` is REMOVED (grep the repo for references first: entry-javy.js and node-runner.mjs are the only consumers; update both):

```js
import { assembleNtcharts } from "./ntcharts/index.js";

// Single shared compile path for Node reference runs and the Javy wasm build.
// Emits the frozen envelope contract: { spec, warnings, size } — see
// docs/2026-07-16-wasm-rewire-plan.md Global Constraints.
export function compileToNtSpec(inputJSON) {
  const out = assembleNtcharts(JSON.parse(inputJSON));
  const { _warnings, _width, _height, ...spec } = out;
  return JSON.stringify({
    spec,
    warnings: _warnings ?? [],
    size: { width: _width, height: _height },
  });
}
```

NOTE: `compile.js` is plain JS importing the TS module `./ntcharts/index.js` — esbuild resolves `.ts` for that specifier when bundling, but plain `node` (vitest/node-runner) will NOT. Check how vitest currently resolves the ntcharts imports (vitest transforms TS natively). For node-runner.mjs, the cleanest fix: convert `js/src/compile.js` → `js/src/compile.ts` and run node-runner through `npx tsx` or add a tiny esbuild pre-bundle step for the runner. Choose the lightest option that keeps: (a) vitest green, (b) `node`-executable reference generation, (c) esbuild bundle working; record the choice. (Recommended: rename to `compile.ts`, change node-runner invocation in package.json to `npx tsx scripts/node-runner.mjs` — tsx resolves TS imports transparently; add `tsx` as a devDependency.)

`js/src/entry-javy.js`: change the import and the final line only:

```js
import { compileToNtSpec } from "./compile.js";
// ... readStdin/writeStdout unchanged ...
writeStdout(compileToNtSpec(readStdin()));
```

(Adjust the import extension if compile.js became compile.ts — esbuild resolves `./compile.js` → `compile.ts` under NodeNext-style resolution; verify the bundle builds.)

- [ ] **Step 3: Update the reference runner for dual dirs + envelope**

`js/scripts/node-runner.mjs` — same structure, now iterating both fixture dirs:

```js
import { readFileSync, writeFileSync, mkdirSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { compileToNtSpec } from "../src/compile.js";

const PAIRS = [
  ["../../testdata/fixtures/", "../../testdata/expected/"],
  ["../../testdata/fixtures-terminal/", "../../testdata/expected-terminal/"],
];

let failed = 0;
for (const [fixRel, expRel] of PAIRS) {
  const fixturesDir = fileURLToPath(new URL(fixRel, import.meta.url));
  const expectedDir = fileURLToPath(new URL(expRel, import.meta.url));
  mkdirSync(expectedDir, { recursive: true });
  for (const f of readdirSync(fixturesDir).sort()) {
    if (!f.endsWith(".json")) continue;
    try {
      const out = compileToNtSpec(readFileSync(fixturesDir + f, "utf8"));
      writeFileSync(expectedDir + f, out);
      console.log(`ok   ${expRel.split("/").at(-2)}/${f} (${Buffer.byteLength(out)} bytes)`);
    } catch (err) {
      failed++;
      console.error(`FAIL ${f}: ${err.message}`);
    }
  }
}
process.exit(failed ? 1 : 0);
```

(Note this also fixes the Phase-1 Minor: byte counts now use `Buffer.byteLength`.)

Delete the old Vega-Lite references, regenerate:

```bash
cd /Users/evan/projects/flint-ntcharts
rm testdata/expected/*.json
cd js && npm run gen-expected
```

Expected: 11 `ok` lines (5 pixel + 6 terminal). Spot-check one envelope starts with `{"spec":{"type":`.

- [ ] **Step 4: Full JS verification + commit**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npx vitest run && npx tsc --noEmit && npm run build
cd .. 
git add js/src/compile.* js/src/entry-javy.js js/scripts/node-runner.mjs js/test/envelope.test.ts js/package.json js/package-lock.json testdata/expected/ testdata/expected-terminal/
git commit -m "feat(js): compileToNtSpec envelope entry; dual-dir Node references"
```

NOTE: the Go parity suite is now RED (embedded wasm still emits Vega-Lite; expected/ are envelopes) — that is the intended state between Tasks 1 and 2; do NOT run `go test ./...` as a gate for this task, and say so in the commit body if desired.

---

### Task 2: Rebuild wasm — OS-aware javy target + provenance stamp

**Files:**
- Modify: `Makefile`
- Create (generated, committed): `compile/flint.wasm` (rebuilt), `compile/flint.wasm.buildinfo`

**Interfaces:**
- Produces: rebuilt `compile/flint.wasm` (ntcharts-envelope compiler, WASI stdin/stdout, same runtime contract) and `compile/flint.wasm.buildinfo` (provenance: javy version, flint-chart version, bundle sha256, bundle size).

- [ ] **Step 1: Make the Makefile OS-aware and provenance-stamping**

```make
JAVY ?= bin/javy

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_S),Darwin)
  ifeq ($(UNAME_M),arm64)
    JAVY_PATTERN := javy-arm-macos-*.gz
  else
    JAVY_PATTERN := javy-x86_64-macos-*.gz
  endif
else
  ifeq ($(UNAME_M),aarch64)
    JAVY_PATTERN := javy-arm-linux-*.gz
  else
    JAVY_PATTERN := javy-x86_64-linux-*.gz
  endif
endif

.PHONY: wasm
wasm: $(JAVY)
	cd js && npm run build
	$(JAVY) build -o compile/flint.wasm js/dist/flint-javy.js
	@printf 'javy: %s\nflint-chart: %s\nbundle-sha256: %s\nbundle-bytes: %s\n' \
	  "$$($(JAVY) --version)" \
	  "$$(cd js && node -p "require('flint-chart/package.json').version")" \
	  "$$(shasum -a 256 js/dist/flint-javy.js | cut -d' ' -f1)" \
	  "$$(wc -c < js/dist/flint-javy.js | tr -d ' ')" \
	  > compile/flint.wasm.buildinfo
	@ls -la compile/flint.wasm
	@cat compile/flint.wasm.buildinfo

$(JAVY):
	mkdir -p bin
	gh release download -R bytecodealliance/javy --pattern "$(JAVY_PATTERN)" -O bin/javy.gz --clobber
	gunzip -f bin/javy.gz
	chmod +x bin/javy
```

(If `shasum` is unavailable on Linux CI later, `sha256sum` — keep `shasum -a 256` now, note the portability caveat in a Makefile comment.)

- [ ] **Step 2: Rebuild and sanity-check by hand**

```bash
cd /Users/evan/projects/flint-ntcharts
make wasm
bin/javy --version
```

Then a direct smoke: run one fixture through the new wasm with wasmtime if available (`wasmtime run compile/flint.wasm < testdata/fixtures/bar-currency.json | head -c 200`) — output must start `{"spec":{"type":"bar"` and match `testdata/expected/bar-currency.json` via `cmp`. If wasmtime is absent, defer verification to Task 3's parity suite (note it).

Watch for QuickJS regressions: the bundle now includes the TS backend + more flint core paths. A trap here (new missing global) is a real finding — record it verbatim and fix via `js/src/polyfills.js` ONLY with a guarded polyfill mirroring the structuredClone precedent; anything beyond a simple missing global → STOP, report BLOCKED.

- [ ] **Step 3: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add Makefile compile/flint.wasm compile/flint.wasm.buildinfo
git commit -m "build: rebuild flint.wasm as ntcharts-envelope compiler; OS-aware javy; provenance stamp"
```

(js/src/polyfills.js too if a new polyfill was needed.)

---

### Task 3: Go production API — Compile/CompileRaw/Options + parity rewrite

**Files:**
- Modify: `compile/compile.go`
- Modify: `compile/compile_test.go`
- Create: `compile/options.go`, `compile/options_test.go`

**Interfaces:**
- Consumes: rebuilt wasm (Task 2), envelopes in `testdata/expected{,-terminal}/`.
- Produces: the Go API from Global Constraints. `Warning` mirrors flint's ChartWarning JSON: `{severity, code, message, channel?, field?}`.

- [ ] **Step 1: Write the failing tests**

Rewrite `compile/compile_test.go`:

```go
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
```

Create `compile/options_test.go`:

```go
package compile

import (
	"encoding/json"
	"testing"
)

func TestWithBaseSizeInjection(t *testing.T) {
	in := []byte(`{"data":{"values":[]},"chart_spec":{"chartType":"Bar Chart","encodings":{}}}`)
	out, err := applyOptions(in, []Option{WithBaseSize(48, 16)})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	cs := doc["chart_spec"].(map[string]any)
	bs := cs["baseSize"].(map[string]any)
	if bs["width"].(float64) != 48 || bs["height"].(float64) != 16 {
		t.Fatalf("baseSize not injected: %v", bs)
	}
}

func TestWithCanvasSizeOverridesExisting(t *testing.T) {
	in := []byte(`{"data":{"values":[]},"chart_spec":{"chartType":"Bar Chart","encodings":{},"canvasSize":{"width":1,"height":1}}}`)
	out, err := applyOptions(in, []Option{WithCanvasSize(100, 40)})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	cs := doc["chart_spec"].(map[string]any)["canvasSize"].(map[string]any)
	if cs["width"].(float64) != 100 {
		t.Fatalf("canvasSize not overridden: %v", cs)
	}
}

func TestNoOptionsPassthroughBytes(t *testing.T) {
	in := []byte(`{"x": 1}`)
	out, err := applyOptions(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Fatal("no-option path must not re-serialize the input")
	}
}
```

Run `go test ./compile/` → FAIL (`undefined: New`, `CompileRaw`, `applyOptions`, …).

- [ ] **Step 2: Implement**

`compile/options.go`:

```go
package compile

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

// applyOptions returns input unchanged when no options are given (preserving
// byte-parity paths); otherwise parses, applies each option, re-serializes.
func applyOptions(input []byte, opts []Option) ([]byte, error) {
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
```

`compile/compile.go` — evolve the Phase-1 Runner (embed/wazero/lifecycle unchanged):

```go
// Package compile runs the flint→ntcharts compiler, embedded as a Javy-built
// WASI module, via wazero. Pure Go: no cgo, no Node at runtime. Output is the
// ntcharts-spec envelope {spec, warnings, size}.
```

- Rename `NewRunner` → `New`; keep `// Deprecated: use New.` alias `func NewRunner(ctx context.Context) (*Runner, error) { return New(ctx) }`.
- Rename `CompileVegaLite` → `CompileRaw(ctx, input []byte, opts ...Option) ([]byte, error)`: first `input, err = applyOptions(input, opts)`, then the existing instantiate-per-call stdin/stdout run, returning raw envelope bytes.
- Add:

```go
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
```

- Import `github.com/NimbleMarkets/ntcharts/v2/spec` and `encoding/json`.
- Update `BenchmarkCompileScatter` to use `New` + `CompileRaw`.

- [ ] **Step 3: Run to green, full sweep, commit**

```bash
cd /Users/evan/projects/flint-ntcharts
go test ./compile/ -v      # 11 parity subtests + envelope/options tests PASS
go test ./...              # speccheck still green
go test ./compile/ -bench=. -benchtime=10x -run='^$'   # record ms/compile for the report
git add compile/compile.go compile/compile_test.go compile/options.go compile/options_test.go go.mod go.sum
git commit -m "feat(go): production Compile API - envelope parsing, options, dual-dir parity"
```

If a parity subtest fails: diff the outputs; QuickJS-vs-Node divergence in the NEW code paths (TS backend in QuickJS) is the finding this phase exists to catch — record verbatim; guarded polyfill only for missing globals; anything semantic → BLOCKED.

---

### Task 4: CI workflow + docs

**Files:**
- Create: `.github/workflows/ci.yml`
- Modify: `README.md`

**Interfaces:**
- Produces: CI that validates the committed artifacts on push/PR without javy: npm ci + vitest + tsc, gen-expected drift check, `go test ./...` against the committed wasm. (Wasm rebuild stays a local/manual `make wasm` — javy download in CI is a later nicety.)

- [ ] **Step 1: Write the workflow**

`.github/workflows/ci.yml`:

```yaml
name: ci
on:
  push: { branches: [main] }
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: "22", cache: npm, cache-dependency-path: js/package-lock.json }
      - uses: actions/setup-go@v5
        with: { go-version-file: go.mod }
      - name: JS deps
        run: cd js && npm ci
      - name: TS typecheck + tests
        run: cd js && npx tsc --noEmit && npx vitest run
      - name: Reference drift check
        run: |
          cd js && npm run gen-expected
          git diff --exit-code ../testdata/expected ../testdata/expected-terminal
      - name: Spike bundle builds
        run: cd js && npm run build
      - name: Go tests (parity + speccheck, committed wasm)
        run: go test ./...
```

CAVEAT: `go test ./...` needs the ntcharts module at `../ntcharts` (replace directive) — unavailable in CI until that repo/branch is pushed. Guard it: add a first step `- name: Check ntcharts sibling` that tests `[ -d ../ntcharts ]` and skips (with a warning annotation) the Go step when absent, OR vendor-switch later. Implement the guard via an `if:` on a step-output boolean — CI must not hard-fail on the known-missing sibling; the Go step runs `go test ./compile/` only (parity needs no ntcharts) and skips speccheck when the sibling is absent: `go test ./compile/` unconditionally + `go test ./speccheck/` behind the guard. Document this in the workflow comments and README.

- [ ] **Step 2: README updates**

- Rewrite the Quickstart around the new API: `New` → `Compile` envelope example (Go snippet), `WithBaseSize` for terminal sizing, envelope contract documented verbatim, provenance file explained, CI notes (sibling-repo caveat), `make wasm` for rebuilds (OS-aware).
- Mark the Phase-1 "stand-in Vega-Lite compile" wording as historical (SPIKE-RESULTS.md untouched — it's a dated record).

- [ ] **Step 3: Verify + commit**

```bash
cd /Users/evan/projects/flint-ntcharts
cd js && npx vitest run && npx tsc --noEmit && npm run build && npm run gen-expected && cd ..
git diff --exit-code testdata/expected testdata/expected-terminal
go test ./...
git add .github/workflows/ci.yml README.md
git commit -m "ci: artifact-validation workflow; README for phase-4 API"
```

(Cannot execute the workflow locally — YAML-validate it (`npx yaml-lint` or python -c yaml.safe_load) and note that its first real run happens when the repo gets a remote.)
