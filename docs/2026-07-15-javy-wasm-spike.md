# Javy WASM Spike Implementation Plan (flint → ntcharts, Phase 1)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prove the flint-chart compiler runs correctly inside a Javy-built WASM module executed by wazero from pure Go, producing byte-identical output to Node across fixture charts.

**Architecture:** Bundle flint-chart (using `assembleVegaLite` as the stand-in compile entry point — the ntcharts Stage-3 backend doesn't exist yet) into a single JS file with esbuild, compile it to `flint.wasm` with Javy (QuickJS engine embedded), and execute it from Go via wazero with a stdin/stdout JSON contract. A Node runner using the *same* JS module generates reference outputs; the Go test asserts byte parity.

**Tech Stack:** flint-chart (npm), esbuild, Javy (Bytecode Alliance), Go ≥1.23, wazero v1.x.

**Spec:** `docs/superpowers/specs/2026-07-15-flint-ntcharts-backend-design.md` (§4 and Phasing §1).

## Global Constraints

- Work happens in a **new directory** `/Users/evan/projects/flint-ntcharts` (seed of the future `github.com/NimbleMarkets/flint-ntcharts` repo). The plan file itself lives in the flint-chart clone.
- Go module path: `github.com/NimbleMarkets/flint-ntcharts`; Go ≥ 1.23; wazero is the ONLY Go dependency.
- esbuild target `es2020`, format `iife`, platform `browser` (QuickJS compatibility).
- Parity requirement is **byte-identical** stdout between Node and wasm for every fixture (both paths run the same bundled JS and `JSON.stringify`).
- Platform: macOS arm64 (Javy release binary `javy-arm-macos-*`).
- Go/no-go criteria (record in `SPIKE-RESULTS.md`): all fixtures parity-pass; per-compile latency < 250 ms; wasm size noted. Fallback on failure: shell out to Node (same JSON contract).

---

### Task 1: Repo scaffold + flint-chart export verification

**Files:**
- Create: `/Users/evan/projects/flint-ntcharts/.gitignore`
- Create: `/Users/evan/projects/flint-ntcharts/go.mod`
- Create: `/Users/evan/projects/flint-ntcharts/js/package.json`

**Interfaces:**
- Consumes: `flint-chart` npm package.
- Produces: an npm project where `import { assembleVegaLite } from "flint-chart"` resolves; a verified list of exact chart-type display names used by fixtures in Task 2.

- [ ] **Step 1: Create the repo and scaffold files**

```bash
mkdir -p /Users/evan/projects/flint-ntcharts/js/src /Users/evan/projects/flint-ntcharts/js/scripts
cd /Users/evan/projects/flint-ntcharts
git init
```

`.gitignore`:

```gitignore
node_modules/
js/dist/
bin/
```

`go.mod`:

```
module github.com/NimbleMarkets/flint-ntcharts

go 1.23
```

`js/package.json`:

```json
{
  "name": "@nimblemarkets/flint-ntcharts-spike",
  "private": true,
  "type": "module",
  "scripts": {
    "build": "esbuild src/entry-javy.js --bundle --format=iife --platform=browser --target=es2020 --outfile=dist/flint-javy.js",
    "gen-expected": "node scripts/node-runner.mjs"
  }
}
```

- [ ] **Step 2: Install dependencies**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npm install flint-chart
npm install --save-dev esbuild
```

Expected: both packages install without errors.

- [ ] **Step 3: Verify exports and chart-type names**

```bash
cd /Users/evan/projects/flint-ntcharts/js
node -e "import('flint-chart').then(m => { console.log('exports:', Object.keys(m).join(', ')); const defs = m.vlAllTemplateDefs ?? []; console.log('chart types:', defs.map(d => d.chart).join(' | ')); })"
```

Expected: `exports:` includes `assembleVegaLite`; `chart types:` lists display names. **Record the exact names** for: scatter, bar, line, stacked bar, heatmap. Task 2's fixtures assume `Scatter Plot`, `Bar Chart`, `Line Chart`, `Stacked Bar Chart`, `Heatmap` — if the printed list differs, use the printed names in the fixtures instead. If `vlAllTemplateDefs` is not exported, get names from `npx -y flint-chart-mcp` docs or the flint-chart clone at `/Users/evan/projects/flint-chart/packages/flint-js/src/vegalite/templates/index.ts`.

- [ ] **Step 4: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add -A
git commit -m "chore: scaffold flint-ntcharts spike (npm + go.mod)"
```

---

### Task 2: Compile core, fixtures, and Node reference outputs

**Files:**
- Create: `js/src/compile.js`
- Create: `js/scripts/node-runner.mjs`
- Create: `testdata/fixtures/scatter.json`, `bar-currency.json`, `line-temporal.json`, `stacked-bar.json`, `heatmap.json`
- Create (generated): `testdata/expected/*.json`

**Interfaces:**
- Consumes: `assembleVegaLite` from `flint-chart`.
- Produces: `compileToVegaLite(inputJSON: string): string` in `js/src/compile.js` (consumed by Task 3's Javy entry and by the Node runner); committed `testdata/fixtures/*.json` and `testdata/expected/*.json` (consumed by Task 5's Go parity test).

- [ ] **Step 1: Write the compile core**

`js/src/compile.js`:

```js
import { assembleVegaLite } from "flint-chart";

// Single shared code path for Node reference runs and the Javy wasm build.
// Byte parity between the two depends on both using exactly this function.
export function compileToVegaLite(inputJSON) {
  const input = JSON.parse(inputJSON);
  const spec = assembleVegaLite(input);
  return JSON.stringify(spec);
}
```

- [ ] **Step 2: Write the fixtures**

`testdata/fixtures/scatter.json`:

```json
{
  "data": { "values": [
    {"weight": 2100, "mpg": 31.5, "origin": "Japan"},
    {"weight": 2875, "mpg": 24.0, "origin": "USA"},
    {"weight": 3200, "mpg": 19.2, "origin": "USA"},
    {"weight": 2450, "mpg": 27.8, "origin": "Germany"},
    {"weight": 1980, "mpg": 33.1, "origin": "Japan"},
    {"weight": 3600, "mpg": 16.5, "origin": "USA"}
  ]},
  "semantic_types": {"weight": "Quantity", "mpg": "Quantity", "origin": "Country"},
  "chart_spec": {
    "chartType": "Scatter Plot",
    "encodings": {"x": {"field": "weight"}, "y": {"field": "mpg"}, "color": {"field": "origin"}},
    "baseSize": {"width": 400, "height": 300}
  }
}
```

`testdata/fixtures/bar-currency.json` (stresses number/currency formatting — the Intl risk area):

```json
{
  "data": { "values": [
    {"product": "Widgets", "revenue": 1250000},
    {"product": "Gadgets", "revenue": 872500},
    {"product": "Gizmos", "revenue": 431200},
    {"product": "Doodads", "revenue": 2103000}
  ]},
  "semantic_types": {"revenue": "Price"},
  "chart_spec": {
    "chartType": "Bar Chart",
    "encodings": {"x": {"field": "product"}, "y": {"field": "revenue"}},
    "baseSize": {"width": 400, "height": 300}
  }
}
```

`testdata/fixtures/line-temporal.json` (stresses date parsing/conversion):

```json
{
  "data": { "values": [
    {"date": "2026-01-01", "value": 104.2},
    {"date": "2026-02-01", "value": 108.9},
    {"date": "2026-03-01", "value": 101.4},
    {"date": "2026-04-01", "value": 115.7},
    {"date": "2026-05-01", "value": 119.3},
    {"date": "2026-06-01", "value": 112.8}
  ]},
  "chart_spec": {
    "chartType": "Line Chart",
    "encodings": {"x": {"field": "date"}, "y": {"field": "value"}},
    "baseSize": {"width": 400, "height": 300}
  }
}
```

`testdata/fixtures/stacked-bar.json`:

```json
{
  "data": { "values": [
    {"quarter": "Q1", "region": "EMEA", "sales": 320},
    {"quarter": "Q1", "region": "APAC", "sales": 210},
    {"quarter": "Q2", "region": "EMEA", "sales": 380},
    {"quarter": "Q2", "region": "APAC", "sales": 260},
    {"quarter": "Q3", "region": "EMEA", "sales": 295},
    {"quarter": "Q3", "region": "APAC", "sales": 310}
  ]},
  "chart_spec": {
    "chartType": "Stacked Bar Chart",
    "encodings": {"x": {"field": "quarter"}, "y": {"field": "sales"}, "color": {"field": "region"}},
    "baseSize": {"width": 400, "height": 300}
  }
}
```

`testdata/fixtures/heatmap.json`:

```json
{
  "data": { "values": [
    {"day": "Mon", "hour": "09", "count": 12},
    {"day": "Mon", "hour": "12", "count": 30},
    {"day": "Mon", "hour": "15", "count": 22},
    {"day": "Tue", "hour": "09", "count": 8},
    {"day": "Tue", "hour": "12", "count": 41},
    {"day": "Tue", "hour": "15", "count": 17},
    {"day": "Wed", "hour": "09", "count": 15},
    {"day": "Wed", "hour": "12", "count": 27},
    {"day": "Wed", "hour": "15", "count": 33}
  ]},
  "chart_spec": {
    "chartType": "Heatmap",
    "encodings": {"x": {"field": "hour"}, "y": {"field": "day"}, "color": {"field": "count"}},
    "baseSize": {"width": 400, "height": 300}
  }
}
```

Use the chart-type names recorded in Task 1 Step 3 if they differ from the above.

- [ ] **Step 3: Write the Node reference runner**

`js/scripts/node-runner.mjs`:

```js
import { readFileSync, writeFileSync, mkdirSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { compileToVegaLite } from "../src/compile.js";

const fixturesDir = fileURLToPath(new URL("../../testdata/fixtures/", import.meta.url));
const expectedDir = fileURLToPath(new URL("../../testdata/expected/", import.meta.url));
mkdirSync(expectedDir, { recursive: true });

let failed = 0;
for (const f of readdirSync(fixturesDir).sort()) {
  if (!f.endsWith(".json")) continue;
  try {
    const out = compileToVegaLite(readFileSync(fixturesDir + f, "utf8"));
    writeFileSync(expectedDir + f, out);
    console.log(`ok   ${f} (${out.length} bytes)`);
  } catch (err) {
    failed++;
    console.error(`FAIL ${f}: ${err.message}`);
  }
}
process.exit(failed ? 1 : 0);
```

- [ ] **Step 4: Run it and verify all fixtures compile**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npm run gen-expected
```

Expected: five `ok <name>.json (...)` lines, exit 0. If a fixture FAILs on an unknown chart type or semantic type, fix the fixture per Task 1 Step 3's recorded names and re-run. Spot-check one output is a plausible Vega-Lite spec:

```bash
head -c 400 ../testdata/expected/scatter.json
```

Expected: JSON starting with Vega-Lite keys (e.g. `"$schema"`, `"data"`, `"mark"`, or `"layer"`).

- [ ] **Step 5: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add -A
git commit -m "feat: compile core, fixtures, and Node reference outputs"
```

---

### Task 3: Javy entry point + esbuild bundle + QuickJS feature audit

**Files:**
- Create: `js/src/entry-javy.js`
- Create (generated, gitignored): `js/dist/flint-javy.js`

**Interfaces:**
- Consumes: `compileToVegaLite` from `js/src/compile.js` (Task 2).
- Produces: `js/dist/flint-javy.js` — a self-contained IIFE bundle whose runtime contract is: read `ChartAssemblyInput` JSON on stdin (fd 0), write spec JSON to stdout (fd 1), throw (non-zero exit) on error. Consumed by Task 4's Javy build.

- [ ] **Step 1: Write the Javy entry point**

`js/src/entry-javy.js`:

```js
import { compileToVegaLite } from "./compile.js";

// Javy exposes Javy.IO for WASI stdin/stdout access.
function readStdin() {
  const chunks = [];
  let total = 0;
  while (true) {
    const buf = new Uint8Array(4096);
    const n = Javy.IO.readSync(0, buf);
    if (n < 0) throw new Error("error reading stdin");
    if (n === 0) break;
    chunks.push(buf.subarray(0, n));
    total += n;
  }
  const all = new Uint8Array(total);
  let off = 0;
  for (const c of chunks) { all.set(c, off); off += c.length; }
  return new TextDecoder().decode(all);
}

function writeStdout(str) {
  const bytes = new TextEncoder().encode(str);
  let off = 0;
  while (off < bytes.length) {
    const n = Javy.IO.writeSync(1, bytes.subarray(off));
    off += (typeof n === "number" && n > 0) ? n : bytes.length - off;
  }
}

writeStdout(compileToVegaLite(readStdin()));
```

- [ ] **Step 2: Build the bundle**

```bash
cd /Users/evan/projects/flint-ntcharts/js
npm run build
ls -la dist/flint-javy.js
```

Expected: bundle builds with no errors (esbuild will fail loudly if flint-chart pulls in Node builtins — if so, record which imports and stop for guidance). Note the bundle size.

- [ ] **Step 3: Audit the bundle for QuickJS-risky features**

```bash
cd /Users/evan/projects/flint-ntcharts/js
grep -cE "Intl\." dist/flint-javy.js; grep -nE "Intl\." dist/flint-javy.js | head -20
grep -nE "toLocale(String|DateString|TimeString)|structuredClone|WeakRef|FinalizationRegistry" dist/flint-javy.js | head -20
```

Expected: ideally zero hits. Record every hit (count + context) — these are the candidate failure points if the parity test (Task 5) fails. Zero hits means low risk; hits in reachable formatting paths mean the parity fixtures (especially `bar-currency` and `line-temporal`) will surface them.

- [ ] **Step 4: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add -A
git commit -m "feat: Javy stdin/stdout entry and esbuild bundle"
```

---

### Task 4: Build flint.wasm with Javy

**Files:**
- Create: `Makefile`
- Create (downloaded, gitignored): `bin/javy`
- Create (generated, committed for the spike): `compile/flint.wasm`

**Interfaces:**
- Consumes: `js/dist/flint-javy.js` (Task 3).
- Produces: `compile/flint.wasm` — a WASI command module (`_start` entry): stdin → spec JSON on stdout. Consumed by Task 5's `go:embed`.

- [ ] **Step 1: Install Javy from GitHub releases**

```bash
cd /Users/evan/projects/flint-ntcharts
mkdir -p bin compile
gh release download -R bytecodealliance/javy --pattern "javy-arm-macos-*.gz" -O bin/javy.gz --clobber
gunzip -f bin/javy.gz
chmod +x bin/javy
bin/javy --version
```

Expected: prints a Javy version (v5+ current as of mid-2026). If the release asset pattern doesn't match, run `gh release view -R bytecodealliance/javy` and use the macOS arm asset name shown.

- [ ] **Step 2: Write the Makefile**

`Makefile`:

```make
JAVY ?= bin/javy

.PHONY: wasm
wasm:
	cd js && npm run build
	$(JAVY) build -o compile/flint.wasm js/dist/flint-javy.js
	@ls -la compile/flint.wasm
```

- [ ] **Step 3: Build and record the size**

```bash
cd /Users/evan/projects/flint-ntcharts
make wasm
```

Expected: `compile/flint.wasm` exists. Record its size for `SPIKE-RESULTS.md` (expect roughly 1–3 MB). If `javy build` rejects the bundle (syntax QuickJS can't parse), record the error verbatim and stop for guidance — that's a go/no-go data point, not something to work around silently.

- [ ] **Step 4: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add -A
git commit -m "build: Javy toolchain and flint.wasm artifact"
```

---

### Task 5: Go compile package with parity test (TDD)

**Files:**
- Create: `compile/compile_test.go`
- Create: `compile/compile.go`
- Test: `compile/compile_test.go`

**Interfaces:**
- Consumes: `compile/flint.wasm` (Task 4), `testdata/fixtures/*.json` + `testdata/expected/*.json` (Task 2).
- Produces: `package compile` with `NewRunner(ctx context.Context) (*Runner, error)`, `(*Runner) CompileVegaLite(ctx context.Context, input []byte) ([]byte, error)`, `(*Runner) Close(ctx context.Context) error`. Task 6 benchmarks against this exact API; later phases keep it.

- [ ] **Step 1: Add wazero and write the failing parity test**

```bash
cd /Users/evan/projects/flint-ntcharts
go get github.com/tetratelabs/wazero@latest
```

`compile/compile_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/evan/projects/flint-ntcharts
go test ./compile/ -v
```

Expected: FAIL — compile errors: `undefined: NewRunner` (package has no non-test source yet).

- [ ] **Step 3: Write the implementation**

`compile/compile.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd /Users/evan/projects/flint-ntcharts
go test ./compile/ -v
```

Expected: PASS with five subtests. **If a subtest fails on output mismatch or a wasm-side exception:** this is the spike doing its job. Diff the outputs (`diff <(echo "$got") <(echo "$want")` on saved files), correlate with the Task 3 Step 3 audit hits, and record the exact divergence in `SPIKE-RESULTS.md` — do not massage the fixture to hide it. Common causes: `Intl`-dependent formatting, locale-dependent `Date` parsing, number `toString` precision differences.

- [ ] **Step 5: Commit**

```bash
cd /Users/evan/projects/flint-ntcharts
git add -A
git commit -m "feat: wazero-based compile package with Node parity test"
```

---

### Task 6: Benchmark + spike report (go/no-go)

**Files:**
- Modify: `compile/compile_test.go` (append benchmark)
- Create: `SPIKE-RESULTS.md`

**Interfaces:**
- Consumes: `Runner` API (Task 5).
- Produces: `SPIKE-RESULTS.md` — the go/no-go record the design spec's Phase 1 gate requires.

- [ ] **Step 1: Append the benchmark to `compile/compile_test.go`**

```go
func BenchmarkCompileScatter(b *testing.B) {
	ctx := context.Background()
	r, err := NewRunner(ctx)
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
		if _, err := r.CompileVegaLite(ctx, input); err != nil {
			b.Fatal(err)
		}
	}
}
```

- [ ] **Step 2: Run the benchmark**

```bash
cd /Users/evan/projects/flint-ntcharts
go test ./compile/ -bench=. -benchtime=10x -run=^$ -v
```

Expected: completes; record ns/op → ms per compile. Gate: < 250 ms per compile.

- [ ] **Step 3: Write `SPIKE-RESULTS.md`**

Structure (fill every value from actual measured output — no estimates):

```markdown
# Javy WASM Spike Results (2026-07-XX)

**Verdict: GO / NO-GO** (one line why)

| Metric | Value | Gate |
| --- | --- | --- |
| Fixtures parity-passing | N / 5 | 5 / 5 |
| Per-compile latency | X ms | < 250 ms |
| flint.wasm size | X MB | informational |
| JS bundle size | X KB | informational |
| Javy version | vX.Y.Z | — |
| QuickJS-risk audit hits (Intl/toLocale/etc.) | N (list) | informational |

## Divergences found
(exact fixture + diff excerpt for any parity failure, or "none")

## Notes for Phase 2+
(anything learned that changes the design spec, or "none")
```

- [ ] **Step 4: Run the full suite one last time and commit**

```bash
cd /Users/evan/projects/flint-ntcharts
go test ./... && cd js && npm run gen-expected && cd ..
git add -A
git commit -m "docs: spike results and compile benchmark"
```

Expected: tests pass, gen-expected produces no git diff (outputs stable).
