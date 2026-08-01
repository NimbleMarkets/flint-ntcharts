# flint → ntcharts — test & verify runbook

A step-by-step to confirm every part works. Split into **automated checks** (copy-paste,
compare output) and two **interactive checks** that need a real terminal / browser.

All commands below were run on 2026-07-17 and the "expect" output is real, not illustrative.

**Paths assumed** (adjust if yours differ):

| repo | path | branch |
|---|---|---|
| flint-ntcharts (hub) | `~/projects/flint-ntcharts` | `main` |
| ntcharts (spec) | `~/projects/ntcharts` | `spec` |
| booba-shim | `~/projects/booba-shim` | `flintchart` |

> **Protected files — leave them alone.** `ntcharts` has pre-existing uncommitted changes to
> `go.mod` / `go.sum` / `go.work.sum`, and `booba-shim` has an uncommitted `ci.yml` edit. `git status`
> showing those after a test run is expected — the tests don't touch them.

---

## 0 · One-shot: is everything green?

Run all three suites back to back. If every line says `ok`, you're done — sections 1–6 are the
breakdown of what this covered.

```bash
cd ~/projects/flint-ntcharts && go test ./... -count=1
cd ~/projects/ntcharts        && go test ./spec/ -count=1
cd ~/projects/booba-shim      && go test ./... -count=1
```

**Expect:** every package prints `ok` (or `[no test files]` for the two doc-only booba packages).

---

## 1 · The hub — native compile path (flint-ntcharts)

```bash
cd ~/projects/flint-ntcharts
go test ./... -count=1
```

**Expect:**

```
ok  github.com/NimbleMarkets/flint-ntcharts/cmd/flint-tui  0.4s
ok  github.com/NimbleMarkets/flint-ntcharts/compile        2.3s
ok  github.com/NimbleMarkets/flint-ntcharts/envelope       0.8s
ok  github.com/NimbleMarkets/flint-ntcharts/speccheck      1.7s
ok  github.com/NimbleMarkets/flint-ntcharts/tui            6.2s
```

What each package proves:

- **compile** — the embedded WASM compiler runs under wazero and its output is **byte-identical to
  Node** across all 13 fixtures. See the parity subtests explicitly:
  ```bash
  go test ./compile/ -run TestParityWithNode -v -count=1 | grep -c PASS   # → 14  (13 fixtures + parent)
  ```
- **speccheck** — every emitted spec is fed through the **real ntcharts renderer** (`spec.Build` + `View()`)
  and must produce a non-empty chart. This is the cross-repo gate — a wrong-but-valid spec fails here.
  ```bash
  go test ./speccheck/ -run TestGoldens -v -count=1 | grep -c -- '--- PASS'   # → 13
  ```
- **envelope** — the `{spec, warnings, size}` / `{error:{…}}` contract parses correctly.
- **tui** — the live-window model, sources (file/stdin/socket), and render loop (race-checked).
- **cmd/flint-tui** — CLI flag parsing and a headless program smoke test.

```bash
go vet ./... && echo "vet clean"
```

---

## 2 · Draw an actual chart from the Go API

The suites assert charts render; this lets you *see* one. Drop a tiny program in a folder inside the repo:

```bash
cd ~/projects/flint-ntcharts
mkdir -p render && cat > render/main.go <<'EOF'
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

const in = `{"data":{"values":[
  {"m":"Jan","v":120},{"m":"Feb","v":180},{"m":"Mar","v":95},{"m":"Apr","v":210}
]},"chart_spec":{"chartType":"Bar Chart",
  "encodings":{"x":{"field":"m"},"y":{"field":"v"}}}}`

func main() {
	ctx := context.Background()
	r, err := compile.New(ctx)          // boots the embedded wasm compiler
	if err != nil { log.Fatal(err) }
	defer r.Close(ctx)

	s, warns, err := r.Compile(ctx, []byte(in), compile.WithBaseSize(44, 12))
	if err != nil { log.Fatal(err) }    // size is in terminal cells

	m, err := spec.Build(s)             // s.Type dispatches to the ntcharts model
	if err != nil { log.Fatal(err) }
	fmt.Println(m.(interface{ View() string }).View())
	fmt.Printf("compiled ok — %d warning(s)\n", len(warns))
}
EOF
go run ./render
rm -rf render     # clean up when done
```

**Expect** a bar chart in block runes (Apr tallest, Mar shortest), then `compiled ok — 0 warning(s)`:

```
                                 ██████████
           ▅▅▅▅▅▅▅▅▅▅            ██████████
           ██████████            ██████████
▆▆▆▆▆▆▆▆▆▆ ██████████            ██████████
██████████ ██████████ ▄▄▄▄▄▄▄▄▄▄ ██████████
██████████ ██████████ ██████████ ██████████
██████████ ██████████ ██████████ ██████████
──────────────────────────────────────────
Jan        Feb        Mar        Apr
```

Swap `"Bar Chart"` for `"Candlestick Chart"` / `"Sparkline"` / `"Heatmap"` etc. to exercise other types.

### …and visualize the same spec as a graphical web chart

The compiled `spec.Spec` carries a **second render surface**: `ToECharts()` turns it into an interactive
Apache ECharts chart you can open in a browser — same spec, no re-compile. Add these lines before the
end of `main()`:

```go
import "io"; import "os"   // add to the import block

// …after spec.Build/View above:
web, err := s.ToECharts()                       // the SAME compiled spec
if err != nil { log.Fatal(err) }
f, _ := os.Create("/tmp/flint-chart.html")
defer f.Close()
web.(interface{ Render(io.Writer) error }).Render(f)
fmt.Println(">> wrote /tmp/flint-chart.html")
```

Then:

```bash
go run ./render && open /tmp/flint-chart.html   # macOS; use xdg-open on Linux
```

**Expect:** the block-rune chart in the terminal *and* a self-contained HTML page that renders an
interactive bar chart in the browser (hover tooltips, the works). One compile, two visualizations —
terminal and web.

> **Caveat:** the web surface (`ToECharts`) is fully wired for **Bar** and **Timeseries** today; the
> other types return a clear "not yet implemented" error there. The terminal surface (`Build`) covers
> all eight. (Rendering the full set to the web is a natural follow-up — the spec already carries the data.)

---

## 3 · The JavaScript backend (flint-ntcharts/js)

```bash
cd ~/projects/flint-ntcharts/js
npx vitest run          # the TS Stage-3 backend, format/colormap, golden emission
npx tsc --noEmit        # type-check
```

**Expect:** `Test Files 8 passed (8) / Tests 40 passed (40)`, then a clean `tsc` (no output).

**Integrity — the golden references are reproducible (no drift):**

```bash
npm run gen-expected
cd .. && git diff --exit-code testdata/expected testdata/expected-terminal && echo "no drift"
```

**Expect:** `no drift` (regenerating the Node references changes nothing).

---

## 4 · Provenance — the committed artifacts match their sources

The embedded wasm and the vendored browser bundle each carry a sha stamp. Confirm they match reality
(read-only; no rebuild needed):

```bash
cd ~/projects/flint-ntcharts
grep wasm-sha256 compile/flint.wasm.buildinfo
shasum -a 256 compile/flint.wasm | cut -d' ' -f1
```

**Expect:** the two hashes are identical (e.g. `8e839bbb6909…`).

```bash
cd ~/projects/booba-shim
grep sha256 web/flintchart/PROVENANCE.txt
shasum -a 256 web/flintchart/flintchart-shim.js cmd/booba-shim-assets/assets/flintchart/flintchart-shim.js
```

**Expect:** the PROVENANCE sha and **both** vendored copies match (e.g. `2c11d949…`). This same
check runs automatically as `booba-shim/flintchart`'s `TestVendoredBundlesMatchProvenance`.

> Optional deeper check (needs the `javy` CLI in `bin/`): `make wasm` and `make browser-shim`
> rebuild the artifacts and must reproduce the same shas from a clean tree.

---

## 5 · The ntcharts renderer (ntcharts, spec branch)

```bash
cd ~/projects/ntcharts
go test ./spec/ -count=1
go vet ./spec/ && echo "vet clean"
```

**Expect:** `ok  github.com/NimbleMarkets/ntcharts/v2/spec`. This is where `buildBar` … `buildOHLC` /
`buildSparkline` and the runnable `ExampleBuild_*` doc-tests live.

---

## 6 · The browser leg (booba-shim, flintchart branch)

**a. Go bridge + wasm build + vendor integrity:**

```bash
cd ~/projects/booba-shim
go test ./... -count=1                    # includes the vendoring-integrity test
go vet ./...
GOOS=js GOARCH=wasm go build ./...        # the shim compiles for the browser target
```

**Expect:** `ok … /flintchart`, clean vet, clean js build.

**b. The vendored bundle actually compiles a chart (headless, no browser):**

```bash
node examples/flintchart-compile/smoke.mjs
```

**Expect:**

```
version: flint-ntcharts flint-chart@0.2.1
warnings: 0
size: { width: 60, height: 16 }
```

This loads the exact file a browser would and runs a compile through it — proof the browser path
produces the same envelope as native.

---

## 7 · INTERACTIVE — the live TUI (needs a real terminal)

No automated harness has a TTY, so this one is on you (~1 minute).

**Easiest: the split-pane playground.** Edit a spec (with JSON syntax highlighting) on the left, watch
it compile and render on the right, live on every keystroke — no second window needed:

```bash
task demo        # or: go run ./cmd/flint-edit
```

Type in the left pane (start with the seeded bar chart), and the right pane redraws. Change
`"Bar Chart"` to `"Sparkline"` or `"Candlestick Chart"`, break the JSON to see the error land on the
status line while the last good chart stays up, resize the window to watch it re-fit. `ctrl+c` quits.

- Press `alt+2` (or `ctrl+2` in a kitty-protocol terminal): the buffer swaps
  to the Timeseries Line example and the chart re-renders; `alt+3` shows the
  Candlestick. `alt+1` returns to the Bar Chart. Unbound chords (`alt+9`) do
  nothing.

The `flint-tui` checks below exercise the same renderer through the file/stdin/socket **hosting** path
(what agents push to), which the playground doesn't cover.

```bash
cd ~/projects/flint-ntcharts
go build ./cmd/flint-tui

cat > /tmp/chart.json <<'EOF'
{"data":{"values":[{"m":"Jan","v":120},{"m":"Feb","v":180},{"m":"Mar","v":95},{"m":"Apr","v":210}]},
 "chart_spec":{"chartType":"Bar Chart","encodings":{"x":{"field":"m"},"y":{"field":"v"}}}}
EOF

./flint-tui /tmp/chart.json
```

**Check, in order:**

1. A bar chart fills the window with a status line at the bottom (`flint-tui · file:/tmp/chart.json · WxH`).
2. **Edit** `/tmp/chart.json` in another pane (change a value or the `chartType` to `"Line Chart"`) and
   save — the chart **redraws within ~250 ms**.
3. **Resize** the terminal window — the chart **re-fits** to the new size (labels re-thin, not just clipped).
4. Feed it a bad doc (`echo 'nonsense' > /tmp/chart.json`) — an **error appears on the status line and the
   last good chart stays on screen**.
5. Press **`q`** (or `ctrl+c`) — clean exit, terminal restored.

**Socket mode** (agents/scripts pushing updates), in two panes:

```bash
# pane A
./flint-tui --listen /tmp/flint.sock
# pane B
printf '%s' '{"type":"sparkline","width":40,"height":4,"data":{"series":[{"name":"cpu","values":[{"y":1},{"y":5},{"y":2},{"y":8},{"y":3},{"y":9}]}]}}' | nc -U /tmp/flint.sock
```

**Expect:** the sparkline appears in pane A the moment pane B sends. (This also exercises the
**direct ntcharts-spec** input kind — note the top-level `"type"`, no `chart_spec`.)

---

## 8 · INTERACTIVE — the browser example (needs a browser)

```bash
cd ~/projects/booba-shim/examples/flintchart-compile
GOOS=js GOARCH=wasm go build -o app.wasm .
go tool booba-shim-assets . --shim=flintchart     # installs the vendored bundle into ./booba-shim/
python3 -m http.server 8000                        # or: task serve
```

Open **http://localhost:8000** and check:

- The page shows **`compiled ok — 60x16 cells, 0 warning(s)`**.
- Below it, the pretty-printed ntcharts-spec JSON (`"type": "bar"`, a `data.series`, etc.).

This proves Go-WASM in the browser compiled a Flint spec through the vendored bundle — the same
envelope the native and Node paths produce.

---

## What "all green" means

| Layer | Check | Section |
|---|---|---|
| Native compile → render | `flint-ntcharts` Go suite + the render program | 1, 2 |
| WASM ↔ Node byte-parity | `TestParityWithNode` (13/13) | 1 |
| Spec → real renderer | `speccheck` (13/13) | 1 |
| TS backend | vitest (40) + tsc + no-drift | 3 |
| Artifact provenance | sha stamps + vendor test | 4 |
| ntcharts `Build()` | `ntcharts` spec suite | 5 |
| Browser bridge + bundle | booba Go suite + js build + node smoke | 6 |
| Live TUI | resize / edit / error / quit | 7 |
| Browser end-to-end | localhost page | 8 |

Sections 0–6 are fully automated and passed as of this writing. Sections 7–8 are the two things only
a human at a terminal / browser can confirm.
