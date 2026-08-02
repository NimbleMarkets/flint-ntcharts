# Timeseries Readability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Line/scatter charts pin a padded data-fitted y-domain when flint's zero-decision excludes zero, and ntcharts' linechart renders the final x-axis label right-aligned instead of dropping it.

**Architecture:** Task 1 ports the upstream ECharts backend's fitted-domain branch into a shared `pinFittedYDomain` helper called by the line and scatter templates, with regenerated goldens. Task 2 rides the artifact train: Node reference fixtures, wasm rebuild + provenance, full Go verification. Task 3 changes only the final-tick branch of `drawXLabel` in the sibling ntcharts repo (`spec` branch) with a Build+View regression test.

**Tech Stack:** TypeScript (flint-chart npm pkg, vitest), esbuild+Javy wasm pipeline, Go (wazero, ntcharts).

**Spec:** `docs/2026-08-01-timeseries-readability-design.md`

## Global Constraints

- No new dependencies in either repo. `computePaddedDomain` is imported from the existing `flint-chart` package (top-level export).
- The `zero === true → emit.y_axis.min = 0` branch in line/scatter templates stays byte-identical; only the negative case gains behavior.
- Fixture count stays 13 — no drift-guard bump. Goldens for `line-temporal`, `line-numeric`, `scatter` change (both scales where the fixture exists); all other goldens must be byte-identical after regen.
- ntcharts change touches exactly two files: `linechart/linechart.go` and `spec/build_ts_xlabel_test.go` (new). The ntcharts working tree has unrelated uncommitted changes (`Taskfile.yml`, `go.mod`, `go.sum`, `go.work.sum`, `examples/`) — never stage or commit those.
- flint-ntcharts work happens on branch `timeseries-readability` (created in Task 1); ntcharts work is a single commit directly on its `spec` branch.
- End state both repos: full Go suites green (`go test ./...` in flint-ntcharts needs `../ntcharts` sibling — present), vitest green, `go vet` clean, gofmt clean.
- `make wasm` (Task 2) needs `bin/javy` — present from prior builds; if missing it auto-downloads via `gh`.
- The flint-ntcharts working tree carries the user's own uncommitted changes (`LICENSE.txt`, `README.md`, `js/package.json`, `evan.md`): NEVER stage or commit them. Every commit in this plan stages explicit paths only — no `git add -A`, no `git add .`, no `git commit -a`.

---

### Task 1: `pinFittedYDomain` + template wiring + goldens

**Files:**
- Create: `js/src/ntcharts/domain.ts`
- Modify: `js/src/ntcharts/templates/line.ts:21` (the `if (cs.y?.zero?.zero)` line)
- Modify: `js/src/ntcharts/templates/scatter.ts:19` (same pattern)
- Test: `js/test/golden-terminal.test.ts` (line-temporal describe), plus regenerated goldens under `testdata/ntspec-golden/` and `testdata/ntspec-golden-terminal/`

**Interfaces:**
- Consumes: `computePaddedDomain(values: number[], padFraction: number): [number, number] | null` from `flint-chart`; `NtInstantiateContext` fields `emit`, `channelSemantics`, `table` (existing).
- Produces: `pinFittedYDomain(emit: any, cs: any, table: any[]): void` in `js/src/ntcharts/domain.ts` — sets `emit.y_axis.min`/`emit.y_axis.max` or does nothing. Task 2 relies only on the changed golden JSON, not on names.

- [ ] **Step 1: Write the failing property test**

In `js/test/golden-terminal.test.ts`, inside the `describe("golden-terminal: line-temporal", ...)` block, add a second `it`:

```ts
  it("pins a padded, fitted y-domain (zero excluded for price lines)", () => {
    const out = assembleFixture("line-temporal", "terminal");
    // Fixture y values: 104.2..119.3 (min 101.4? no — see below). Data:
    // [104.2, 108.9, 101.4, 115.7, 119.3, 112.8] → min 101.4, max 119.3,
    // span 17.9, pad 5% = 0.895.
    expect(out.y_axis.min).toBeCloseTo(100.505, 3);
    expect(out.y_axis.max).toBeCloseTo(120.195, 3);
  });
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd js && npx vitest run test/golden-terminal.test.ts`
Expected: FAIL — `out.y_axis.min` is `undefined` (template emits no min when zero is excluded).

- [ ] **Step 3: Create `js/src/ntcharts/domain.ts`**

```ts
import { computePaddedDomain } from "flint-chart";

// pinFittedYDomain pins emit.y_axis.min/max to a padded data-fitted domain
// when flint's zero-decision EXCLUDES zero. Without the pin, ntcharts'
// terminal linecharts default their view to a 0 baseline (New() starts at
// 0..1 and auto-range only expands), squashing e.g. a 101..121 price series
// into the top rows. Mirrors the upstream ECharts backend's fitted-domain
// branch (instantiate-spec.ts): !zero && domainPadFraction > 0 →
// computePaddedDomain → axis min/max. The zero===true case is handled at
// the call sites (min = 0), matching upstream.
export function pinFittedYDomain(emit: any, cs: any, table: any[]): void {
  const decision = cs.y?.zero;
  const field = cs.y?.field;
  if (!decision || decision.zero || !(decision.domainPadFraction > 0) || field == null) return;
  const values = table
    .map((row: any) => row[field])
    .filter((v: any) => typeof v === "number" && Number.isFinite(v));
  const padded = computePaddedDomain(values, decision.domainPadFraction);
  if (padded) {
    emit.y_axis.min = padded[0];
    emit.y_axis.max = padded[1];
  }
}
```

- [ ] **Step 4: Wire the templates**

In `js/src/ntcharts/templates/line.ts`, add the import and change line 21:

```ts
import { pinFittedYDomain } from "../domain.js";
```

```ts
    if (cs.y?.zero?.zero) emit.y_axis.min = 0;
    else pinFittedYDomain(emit, cs, table);
```

In `js/src/ntcharts/templates/scatter.ts`, identically: add the same import and change line 19 to the same two-line form:

```ts
import { pinFittedYDomain } from "../domain.js";
```

```ts
    if (cs.y?.zero?.zero) emit.y_axis.min = 0;
    else pinFittedYDomain(emit, cs, table);
```

- [ ] **Step 5: Typecheck and run the failing test to green**

Run: `cd js && npx tsc --noEmit && npx vitest run test/golden-terminal.test.ts`
Expected: the new property test PASSES; the `expectGolden` assertions for line-temporal / line-numeric / scatter FAIL (goldens are stale — that is the next step).

- [ ] **Step 6: Regenerate goldens**

Run: `cd js && UPDATE_GOLDEN=1 npx vitest run && npx vitest run`
Expected: both runs end green. Then verify the blast radius is exactly the fitted-domain charts:

Run: `git status --short testdata/`
Expected: modified files are ONLY `testdata/ntspec-golden/line-temporal.json`, `testdata/ntspec-golden/scatter.json`, `testdata/ntspec-golden-terminal/line-temporal.json`, `testdata/ntspec-golden-terminal/line-numeric.json`, `testdata/ntspec-golden-terminal/scatter.json`. Spot-check one:

Run: `python3 -c "import json; d=json.load(open('testdata/ntspec-golden-terminal/line-temporal.json')); print(d['y_axis'])"`
Expected: `{'title': 'value', 'min': 100.505, 'max': 120.195}` (key order may differ).

If any OTHER golden changed, stop and report BLOCKED — the helper is leaking into charts it must not touch.

- [ ] **Step 7: Commit**

```bash
git add js/src/ntcharts/domain.ts js/src/ntcharts/templates/line.ts js/src/ntcharts/templates/scatter.ts js/test/golden-terminal.test.ts testdata/ntspec-golden/ testdata/ntspec-golden-terminal/
git commit -m "fix(ts): pin padded fitted y-domain when zero excluded (line, scatter)"
```

---

### Task 2: artifact train — reference fixtures, wasm, Go verification

**Files:**
- Modify (generated): `testdata/expected/*.json`, `testdata/expected-terminal/*.json` (only line-temporal / line-numeric / scatter entries)
- Modify (generated): `compile/flint.wasm`, `compile/flint.wasm.buildinfo`

**Interfaces:**
- Consumes: Task 1's committed TS change.
- Produces: committed artifacts the Go suite validates; nothing downstream imports names from this task.

- [ ] **Step 1: Regenerate the Node reference fixtures**

Run: `cd js && npm run gen-expected`
Then: `git status --short testdata/expected testdata/expected-terminal`
Expected: only line-temporal, line-numeric, scatter entries changed (same blast-radius rule as Task 1 Step 6 — any other diff is BLOCKED).

- [ ] **Step 2: Rebuild the wasm module + provenance**

Run: `make wasm` (from the repo root)
Expected: regenerates `compile/flint.wasm` and `compile/flint.wasm.buildinfo` (new `bundle-sha256` / `wasm-sha256`; `flint-chart: 0.2.1` unchanged).

- [ ] **Step 3: Run the full Go verification train**

Run: `go test ./... -count=1 && go vet ./...`
Expected: all 6 packages PASS — parity (13/13 fixtures against the new expected/), speccheck (goldens still Build+render in real ntcharts), and flint-edit's `TestExamplesCompile` (fitted pin emits no warnings, state stays 1). vet clean.

- [ ] **Step 4: Commit**

```bash
git add testdata/expected testdata/expected-terminal compile/flint.wasm compile/flint.wasm.buildinfo
git commit -m "build: regen reference fixtures + wasm for fitted y-domain"
```

---

### Task 3: final x label right-aligned (ntcharts repo)

**Files:**
- Modify: `/Users/evan/projects/ntcharts/linechart/linechart.go:413-439` (`drawXLabel`)
- Test: Create `/Users/evan/projects/ntcharts/spec/build_ts_xlabel_test.go`

**Interfaces:**
- Consumes: nothing from Tasks 1–2 (independent repo).
- Produces: renderer behavior only; no exported names change.

Work in `/Users/evan/projects/ntcharts` on the already-checked-out `spec` branch. The tree has unrelated uncommitted changes (`Taskfile.yml`, `go.mod`, `go.sum`, `go.work.sum`, `examples/`): NEVER stage them; commit exactly the two files below.

- [ ] **Step 1: Write the failing test**

Create `/Users/evan/projects/ntcharts/spec/build_ts_xlabel_test.go`:

```go
// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"strings"
	"testing"
	"time"
)

// TestTimeSeriesFinalXLabel guards the drawXLabel final-tick rule: the label
// under the newest data must render right-aligned when it does not fit
// left-anchored at the axis end, not silently vanish. Six monthly points at
// width 45 previously rendered labels Jan..May with Jun dropped.
func TestTimeSeriesFinalXLabel(t *testing.T) {
	ms := func(mo time.Month) float64 {
		return float64(time.Date(2026, mo, 1, 0, 0, 0, 0, time.UTC).UnixMilli())
	}
	s := Spec{
		Type: ChartTypeTimeSeries, Width: 45, Height: 12,
		Data: Data{Series: []Series{{Name: "px", Values: []DataPoint{
			{X: ms(time.January), Y: 104.2}, {X: ms(time.February), Y: 108.9},
			{X: ms(time.March), Y: 101.4}, {X: ms(time.April), Y: 115.7},
			{X: ms(time.May), Y: 119.3}, {X: ms(time.June), Y: 112.8},
		}}}},
		XAxis: XAxis{Type: "time", Format: Format{Kind: "time", Layout: "Jan"}},
	}
	got, err := Build(s) // Build draws the model; View() is ready immediately
	if err != nil {
		t.Fatalf("Build(timeseries): %v", err)
	}
	view := got.(interface{ View() string }).View()
	if !strings.Contains(view, "May") {
		t.Fatalf("sanity: expected an interior month label in view:\n%s", view)
	}
	if !strings.Contains(view, "Jun") {
		t.Fatalf("final month label missing — last x tick was dropped:\n%s", view)
	}
	// Right-aligned: Jun ends at (or within a cell of) the row's right edge.
	for _, line := range strings.Split(view, "\n") {
		if idx := strings.Index(line, "Jun"); idx >= 0 {
			if idx < len(line)-len("Jun ") {
				t.Fatalf("Jun not right-aligned (idx %d in %q)", idx, line)
			}
		}
	}
}
```

API notes (verified against the checkout): `XAxis.Format` is a value `Format` (not a pointer); `YAxis.Min`/`Max` are `*float64`; `Build` calls `DrawAll` internally so the test reads `View()` directly (same viewer-interface idiom as flint-ntcharts' speccheck). `DataPoint.X` for timeseries is ms-since-epoch as float64. If the sanity assertion fails because tick positions land on different months at this size, adjust width (44–50) until an interior label and a dropped final label reproduce — the RED run must fail specifically on the missing "Jun".

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/evan/projects/ntcharts && go test ./spec/ -run TestTimeSeriesFinalXLabel -count=1`
Expected: FAIL at the `Jun` assertion ("final month label missing"). If it fails earlier (compile error, sanity check), fix the test construction — the defect it must expose is specifically the missing final label.

- [ ] **Step 3: Implement the final-tick fallback in `drawXLabel`**

In `/Users/evan/projects/ntcharts/linechart/linechart.go`, replace the label-set block inside `drawXLabel` (currently lines 423–433):

```go
		// can only set if rune to the left of target coordinates is empty
		if c := m.Canvas.Cell(canvas.Point{X: m.origin.X + i - 1, Y: m.origin.Y + 1}); c.Rune == runes.Null {
			v := m.viewMinX + (increment * float64(i)) // value to set under X axis
			s := m.XLabelFormatter(i, v)
			// dont display if number will be cut off or value repeats
			sLen := len(s) + m.origin.X + i
			if (s != lastVal) && (sLen <= m.Canvas.Width()) {
				m.Canvas.SetStringWithStyle(canvas.Point{X: m.origin.X + i, Y: m.origin.Y + 1}, s, m.LabelStyle)
				lastVal = s
			} else if i == last && s != lastVal {
				// Final tick: the axis-end label does not fit left-anchored.
				// Right-align it into the remaining width instead of dropping
				// it, so the newest data is always labeled — subject to the
				// same guards as any other label (left neighbor empty, no
				// repeat of the previous label).
				if x := m.Canvas.Width() - len(s); x > m.origin.X {
					if m.Canvas.Cell(canvas.Point{X: x - 1, Y: m.origin.Y + 1}).Rune == runes.Null {
						m.Canvas.SetStringWithStyle(canvas.Point{X: x, Y: m.origin.Y + 1}, s, m.LabelStyle)
						lastVal = s
					}
				}
			}
		}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/evan/projects/ntcharts && go test ./spec/ -run TestTimeSeriesFinalXLabel -count=1`
Expected: PASS

- [ ] **Step 5: Run the affected suites in ntcharts**

Run: `cd /Users/evan/projects/ntcharts && go test ./linechart/... ./spec/ -count=1 && go vet ./linechart/... ./spec/ && gofmt -l linechart spec`
Expected: all PASS, vet clean, no gofmt output. If an existing linechart test asserts an exact canvas that now gains a right-aligned final label, inspect the diff: an added label at the right edge is the intended behavior change — update that test's expected output and say so in the report; any other difference is a regression to fix.

- [ ] **Step 6: Cross-repo end-to-end verification**

Run: `cd /Users/evan/projects/flint-ntcharts && go test ./... -count=1`
Expected: all 6 packages PASS against the modified sibling (speccheck exercises the new drawXLabel path).

- [ ] **Step 7: Commit (ntcharts repo, exactly two files)**

```bash
cd /Users/evan/projects/ntcharts
git add linechart/linechart.go spec/build_ts_xlabel_test.go
git commit -m "fix(linechart): right-align final x label instead of dropping it"
git status --short   # Taskfile.yml, go.mod etc. must still show as unstaged
```
