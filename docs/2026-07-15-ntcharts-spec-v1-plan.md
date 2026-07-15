# ntcharts-spec v1 Implementation Plan (flint → ntcharts, Phase 2)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Evolve the experimental `ntcharts/spec` package into ntcharts-spec v1 — axis/format structs, per-chart data shapes, bar orientation/stacking — and implement `Build()` for the tier-1 chart types (line, scatter, heatmap).

**Architecture:** Breaking schema changes are allowed (the format is experimental). `Spec` gains top-level `XAxis`/`YAxis` structs (absorbing the loose axis fields), a Go-native `Format` directive (never d3 syntax), `Heat` data for heatmaps, `OHLCPoint` (schema only in this phase), and `Theme.Gradient`. New `Build()` branches drive: wavelinechart (line), base linechart primitives (scatter — ntcharts has no scatter model), and the heatmap package (with hex-gradient → `[]color.Color` interpolation).

**Tech Stack:** Go 1.25, `github.com/NimbleMarkets/ntcharts/v2` (repo itself), `charm.land/lipgloss/v2`, stdlib `image/color`.

**Spec:** `/Users/evan/projects/flint-ntcharts/docs/2026-07-15-flint-ntcharts-backend-design.md` §1 ("ntcharts-spec v1").

## Global Constraints

- Work in `/Users/evan/projects/ntcharts` on the existing **`spec` branch** (do not create a new branch, do not touch `main`/`v2`).
- **The working tree has uncommitted user changes in `go.mod`, `go.sum`, `examples/go.mod`, `examples/go.sum`. NEVER run `git add -A`, `git add .`, or `git add -u`. Stage only the exact files your task touched, by explicit path. Never commit or revert the four files above.**
- Module is `github.com/NimbleMarkets/ntcharts/v2`; lipgloss import is `charm.land/lipgloss/v2` (NOT github.com/charmbracelet/lipgloss). Follow existing imports in `spec/build.go`.
- Test command: `go test ./spec/ -v`; also `go vet ./spec/` must stay clean. Do not run or modify tests outside `spec/`.
- Design principles that bind every task: every option is optional-with-autoscale (surfaces fill gaps); surfaces are free to IGNORE options they can't support, but must ERROR on data they can't render faithfully (e.g. grouped bars).
- `ToECharts()` (web surface) must keep compiling and its existing bar/timeseries tests passing after the schema migration, but new tier-1 types are NOT extended on the web surface in this phase (`toECharts<Kind>` stays not-implemented for line/scatter/heatmap).
- Chart-type display semantics: `ChartTypeLine` = numeric-X line (wavelinechart), `ChartTypeTimeSeries` = time-X line (existing), `ChartTypeScatter`, `ChartTypeHeatmap`.
- Verified API signatures (do not re-derive): `wavelinechart.New(w, h int, opts ...Option) Model`, `(*Model).PlotDataSet(n string, f canvas.Float64Point)`, `(*Model).SetDataSetStyles(n string, ls runes.LineStyle, s lipgloss.Style)`, `(*Model).DrawAll()`, `wavelinechart.WithYRange(min, max float64)`; `linechart.New(w, h int, minX, maxX, minY, maxY float64, opts ...Option) Model`, `(*Model).DrawXYAxisAndLabel()`, `(*Model).DrawRuneWithStyle(f canvas.Float64Point, r rune, s lipgloss.Style)`, `linechart.WithXLabelFormatter(fmt LabelFormatter)`, `linechart.WithYLabelFormatter(fmt LabelFormatter)`, `linechart.LabelFormatter = func(int, float64) string`; `heatmap.New(w, h int, opts ...Option) Model`, `heatmap.NewHeatPoint(x, y, value float64) HeatPoint`, `(*Model).Push(p HeatPoint)`, `(*Model).PushAllMatrixRow(dataRows [][]float64)`, `(*Model).Draw()`, `heatmap.WithColorScale(cs []color.Color)`, `heatmap.WithValueRange(minVal, maxVal float64)`, `heatmap.WithAutoValueRange()`; `barchart.WithHorizontalBars()`.

---

### Task 1: Schema v1 — new types, Validate, and migration of existing surfaces

**Files:**
- Modify: `spec/spec.go` (types + Validate)
- Modify: `spec/build.go` (field-location migration only)
- Modify: `spec/echarts.go` (field-location migration only)
- Modify: `spec/example_test.go`, `spec/echarts_test.go` (field-location migration only)
- Create: `spec/spec_test.go`

**Interfaces:**
- Produces (all later tasks depend on these exact shapes):

```go
// Format describes how numeric or time values render as labels.
// Kind: "" or "number" (plain), "percent" (v*100 + "%"), "currency"
// (symbol prefix), "si" (k/M/G/T suffix), "time" (ms-since-epoch via Layout).
type Format struct {
	Kind      string `json:"kind,omitempty"`
	Precision *int   `json:"precision,omitempty"` // decimals; nil = kind default
	Currency  string `json:"currency,omitempty"`  // symbol for kind "currency"; default "$"
	Layout    string `json:"layout,omitempty"`    // Go time layout for kind "time"
}

func (f Format) IsZero() bool // true when all fields are zero-valued

type XAxis struct {
	Title  string   `json:"title,omitempty"`
	Type   string   `json:"type,omitempty"` // XAxisCategory | XAxisTime | XAxisValue | ""
	Labels []string `json:"labels,omitempty"`
	Format Format   `json:"format,omitempty"`
}

type YAxis struct {
	Title  string   `json:"title,omitempty"`
	Min    *float64 `json:"min,omitempty"` // nil = autoscale
	Max    *float64 `json:"max,omitempty"`
	Format Format   `json:"format,omitempty"`
}

// Orientation constants for Options.Orientation.
const (
	OrientationVertical   = "vertical"
	OrientationHorizontal = "horizontal"
)

type Options struct {
	ShowLegend  bool   `json:"show_legend,omitempty"`
	ShowGrid    bool   `json:"show_grid,omitempty"`
	Orientation string `json:"orientation,omitempty"` // bar charts; "" = vertical
	Stacked     bool   `json:"stacked,omitempty"`     // bar charts
}

type DataPoint struct {
	X    any      `json:"x,omitempty"`
	Y    float64  `json:"y"`
	Size *float64 `json:"size,omitempty"` // scatter point weight; surfaces may ignore
}

type OHLCPoint struct {
	T any     `json:"t"` // time: time.Time, RFC3339 string, or ms-since-epoch
	O float64 `json:"o"`
	H float64 `json:"h"`
	L float64 `json:"l"`
	C float64 `json:"c"`
}

// Series gains: OHLC []OHLCPoint `json:"ohlc,omitempty"` (rendering lands in a later phase)

type HeatCell struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type HeatData struct {
	Cells    []HeatCell  `json:"cells,omitempty"`
	Matrix   [][]float64 `json:"matrix,omitempty"` // row-major; alternative to Cells
	MinValue *float64    `json:"min_value,omitempty"` // color-scale domain; nil = auto
	MaxValue *float64    `json:"max_value,omitempty"`
}

// Theme gains: Gradient []string `json:"gradient,omitempty"` — ordered hex
// stops ("#rrggbb") for sequential colormaps (heatmap).

// Spec gains: XAxis XAxis `json:"x_axis,omitempty"`, YAxis YAxis `json:"y_axis,omitempty"`,
// Heat *HeatData `json:"heat,omitempty"`.
// REMOVED: Data.XAxisType, Data.XAxisLabels (→ XAxis.Type / XAxis.Labels),
// Options.YAxisMin/YAxisMax (→ YAxis.Min/Max), Options.TimeFormat (→ XAxis.Format{Kind:"time", Layout}).
// Data keeps: Series, XAxisData.
```

- [ ] **Step 1: Write the failing tests**

Create `spec/spec_test.go`:

```go
package spec

import (
	"encoding/json"
	"strings"
	"testing"
)

func f64(v float64) *float64 { return &v }

func TestValidateSchemaV1(t *testing.T) {
	base := Spec{
		Type: ChartTypeBar, Width: 40, Height: 10,
		Data: Data{Series: []Series{{Name: "a", Values: []DataPoint{{Y: 1}}}}},
	}

	t.Run("valid base", func(t *testing.T) {
		if err := base.Validate(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})

	t.Run("bad orientation", func(t *testing.T) {
		s := base
		s.Options.Orientation = "diagonal"
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "orientation") {
			t.Fatalf("expected orientation error, got %v", err)
		}
	})

	t.Run("bad format kind", func(t *testing.T) {
		s := base
		s.YAxis.Format = Format{Kind: "roman-numerals"}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "format") {
			t.Fatalf("expected format-kind error, got %v", err)
		}
	})

	t.Run("heatmap requires heat data", func(t *testing.T) {
		s := Spec{Type: ChartTypeHeatmap, Width: 40, Height: 10}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "Heat") {
			t.Fatalf("expected Heat-required error, got %v", err)
		}
	})

	t.Run("heatmap with cells needs no series", func(t *testing.T) {
		s := Spec{Type: ChartTypeHeatmap, Width: 40, Height: 10,
			Heat: &HeatData{Cells: []HeatCell{{X: 0, Y: 0, Z: 1}}}}
		if err := s.Validate(); err != nil {
			t.Fatalf("expected valid, got %v", err)
		}
	})

	t.Run("ohlc requires ohlc points", func(t *testing.T) {
		s := Spec{Type: ChartTypeOHLC, Width: 40, Height: 10,
			Data: Data{Series: []Series{{Name: "px", Values: []DataPoint{{Y: 1}}}}}}
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "OHLC") {
			t.Fatalf("expected OHLC-required error, got %v", err)
		}
	})
}

func TestSpecJSONRoundTrip(t *testing.T) {
	in := Spec{
		Type: ChartTypeHeatmap, Title: "t", Width: 30, Height: 8,
		XAxis: XAxis{Title: "hour", Type: XAxisValue, Format: Format{Kind: "number", Precision: f64i(0)}},
		YAxis: YAxis{Title: "day", Min: f64(0), Max: f64(6)},
		Heat: &HeatData{
			Cells:    []HeatCell{{X: 0, Y: 1, Z: 3.5}},
			MinValue: f64(0), MaxValue: f64(10),
		},
		Theme: Theme{Gradient: []string{"#000000", "#ff0000"}},
	}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out Spec
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.XAxis.Title != "hour" || out.YAxis.Max == nil || *out.YAxis.Max != 6 ||
		out.Heat == nil || len(out.Heat.Cells) != 1 || out.Heat.Cells[0].Z != 3.5 ||
		len(out.Theme.Gradient) != 2 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
}

func f64i(v int) *int { return &v }

func TestFormatIsZero(t *testing.T) {
	if !(Format{}).IsZero() {
		t.Fatal("zero Format should report IsZero")
	}
	if (Format{Kind: "si"}).IsZero() {
		t.Fatal("non-zero Format must not report IsZero")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -v -run 'TestValidateSchemaV1|TestSpecJSONRoundTrip|TestFormatIsZero'
```

Expected: FAIL — compile errors (`undefined: XAxis`, `undefined: HeatData`, etc.).

- [ ] **Step 3: Implement schema v1 in `spec/spec.go`**

Add the types exactly as specified in the Interfaces block above. Update `Spec`:

```go
type Spec struct {
	Type     ChartType `json:"type"`
	Title    string    `json:"title,omitempty"`
	Subtitle string    `json:"subtitle,omitempty"`
	Width    int       `json:"width"`
	Height   int       `json:"height"`

	XAxis XAxis `json:"x_axis,omitempty"`
	YAxis YAxis `json:"y_axis,omitempty"`

	Data    Data      `json:"data"`
	Heat    *HeatData `json:"heat,omitempty"`
	Options Options   `json:"options,omitempty"`
	Theme   Theme     `json:"theme,omitempty"`
}
```

Remove `Data.XAxisType`, `Data.XAxisLabels`, `Options.YAxisMin`, `Options.YAxisMax`, `Options.TimeFormat`. Add `Series.OHLC`, `DataPoint.Size`, `Theme.Gradient`.

`Format.IsZero`:

```go
func (f Format) IsZero() bool {
	return f.Kind == "" && f.Precision == nil && f.Currency == "" && f.Layout == ""
}
```

Extend `Validate()` — keep existing checks, adjust and add:

```go
var validFormatKinds = map[string]bool{
	"": true, "number": true, "percent": true, "currency": true, "si": true, "time": true,
}

func validateFormat(name string, f Format) error {
	if !validFormatKinds[f.Kind] {
		return fmt.Errorf("spec: %s format kind %q is not one of number|percent|currency|si|time", name, f.Kind)
	}
	return nil
}
```

In `Validate()`:
- The "at least one Series" rule becomes: required for every type EXCEPT `ChartTypeHeatmap` when `Heat` is non-nil.
- `ChartTypeHeatmap`: require `s.Heat != nil` and `len(s.Heat.Cells) > 0 || len(s.Heat.Matrix) > 0`, else `fmt.Errorf("spec: heatmap requires Heat data (cells or matrix)")`.
- `ChartTypeOHLC`: require at least one series with `len(ser.OHLC) > 0`, else `fmt.Errorf("spec: ohlc requires Series.OHLC points")`.
- Orientation: must be `""`, `OrientationVertical`, or `OrientationHorizontal`, else `fmt.Errorf("spec: orientation %q must be vertical or horizontal", ...)`.
- Call `validateFormat("x_axis", s.XAxis.Format)` and `validateFormat("y_axis", s.YAxis.Format)`.

- [ ] **Step 4: Migrate `spec/build.go`, `spec/echarts.go`, and the existing tests**

Mechanical field relocation only — no behavior changes in this task:
- `s.Data.XAxisLabels` → `s.XAxis.Labels`; `s.Data.XAxisType` → `s.XAxis.Type` (wherever referenced).
- `s.Options.YAxisMax` → `s.YAxis.Max`; `s.Options.YAxisMin` → `s.YAxis.Min`.
- `s.Options.TimeFormat` → time-format reads become `s.XAxis.Format.Layout` (keep the existing not-yet-wired comment in build.go accurate).
- Update `spec/example_test.go` and `spec/echarts_test.go` fixtures to the new field locations (same values, same `// Output:` blocks — rendered output must not change).

- [ ] **Step 5: Run the full spec suite**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -v && go vet ./spec/
```

Expected: PASS — new schema tests plus all pre-existing examples/tests green with unchanged golden output.

- [ ] **Step 6: Commit (explicit paths only)**

```bash
cd /Users/evan/projects/ntcharts
git add spec/spec.go spec/spec_test.go spec/build.go spec/echarts.go spec/example_test.go spec/echarts_test.go
git commit -m "feat(spec): schema v1 - axis structs, format directive, heat/ohlc/gradient shapes"
```

---

### Task 2: Format engine

**Files:**
- Create: `spec/format.go`
- Create: `spec/format_test.go`
- Modify: `spec/build.go` (wire formatters into buildTimeSeries)

**Interfaces:**
- Consumes: `Format` (Task 1).
- Produces: `FormatValue(f Format, v float64) string` and `(Format) labelFormatter() linechart.LabelFormatter` (returns nil when `f.IsZero()`), used by Tasks 3–4. buildTimeSeries wires `WithXLabelFormatter`/`WithYLabelFormatter` when formats are set.

- [ ] **Step 1: Write the failing tests**

Create `spec/format_test.go`:

```go
package spec

import "testing"

func TestFormatValue(t *testing.T) {
	p := func(v int) *int { return &v }
	cases := []struct {
		name string
		f    Format
		v    float64
		want string
	}{
		{"default plain", Format{}, 12.5, "12.5"},
		{"number precision", Format{Kind: "number", Precision: p(2)}, 3.14159, "3.14"},
		{"percent default", Format{Kind: "percent"}, 0.42, "42%"},
		{"percent precision", Format{Kind: "percent", Precision: p(1)}, 0.4567, "45.7%"},
		{"currency default", Format{Kind: "currency"}, 1250, "$1250.00"},
		{"currency symbol", Format{Kind: "currency", Currency: "€", Precision: p(0)}, 99.9, "€100"},
		{"si thousands", Format{Kind: "si"}, 12500, "12.5k"},
		{"si millions", Format{Kind: "si", Precision: p(2)}, 3400000, "3.40M"},
		{"si small passthrough", Format{Kind: "si"}, 999, "999.0"},
		{"si negative", Format{Kind: "si"}, -2500000000, "-2.5G"},
		{"time layout", Format{Kind: "time", Layout: "2006-01"}, 1767225600000, "2026-01"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FormatValue(c.f, c.v); got != c.want {
				t.Fatalf("FormatValue(%+v, %v) = %q, want %q", c.f, c.v, got, c.want)
			}
		})
	}
}

func TestLabelFormatterNilWhenZero(t *testing.T) {
	if (Format{}).labelFormatter() != nil {
		t.Fatal("zero Format must produce nil LabelFormatter (chart keeps its default)")
	}
	lf := Format{Kind: "si"}.labelFormatter()
	if lf == nil {
		t.Fatal("non-zero Format must produce a formatter")
	}
	if got := lf(0, 2000); got != "2.0k" {
		t.Fatalf("formatter(2000) = %q, want 2.0k", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -run 'TestFormatValue|TestLabelFormatterNilWhenZero' -v
```

Expected: FAIL — `undefined: FormatValue`.

- [ ] **Step 3: Implement `spec/format.go`**

```go
// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"math"
	"strconv"
	"time"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

// FormatValue renders v according to f. Zero-valued fields fall back to
// per-kind defaults; an empty Kind renders a plain minimal float.
func FormatValue(f Format, v float64) string {
	prec := -1
	if f.Precision != nil {
		prec = *f.Precision
	}
	switch f.Kind {
	case "percent":
		if prec < 0 {
			prec = 0
		}
		return strconv.FormatFloat(v*100, 'f', prec, 64) + "%"
	case "currency":
		if prec < 0 {
			prec = 2
		}
		sym := f.Currency
		if sym == "" {
			sym = "$"
		}
		return sym + strconv.FormatFloat(v, 'f', prec, 64)
	case "si":
		return formatSI(v, prec)
	case "time":
		layout := f.Layout
		if layout == "" {
			layout = "2006-01-02"
		}
		return time.UnixMilli(int64(v)).UTC().Format(layout)
	default: // "" or "number"
		return strconv.FormatFloat(v, 'f', prec, 64)
	}
}

var siSuffixes = []struct {
	factor float64
	suffix string
}{
	{1e12, "T"},
	{1e9, "G"},
	{1e6, "M"},
	{1e3, "k"},
}

func formatSI(v float64, prec int) string {
	if prec < 0 {
		prec = 1
	}
	abs := math.Abs(v)
	for _, s := range siSuffixes {
		if abs >= s.factor {
			return strconv.FormatFloat(v/s.factor, 'f', prec, 64) + s.suffix
		}
	}
	return strconv.FormatFloat(v, 'f', prec, 64)
}

// labelFormatter bridges a Format to the linechart label-formatter hook.
// A zero Format returns nil so charts keep their own defaults.
func (f Format) labelFormatter() linechart.LabelFormatter {
	if f.IsZero() {
		return nil
	}
	return func(_ int, v float64) string { return FormatValue(f, v) }
}
```

- [ ] **Step 4: Wire formatters into buildTimeSeries**

In `spec/build.go`, in `buildTimeSeries`, where options are assembled before `timeserieslinechart.New`: check `timeserieslinechart`'s own options file for `WithXLabelFormatter`/`WithYLabelFormatter` (they exist per `linechart/timeserieslinechart/options.go:46/53`). Add:

```go
if yf := s.YAxis.Format.labelFormatter(); yf != nil {
	opts = append(opts, timeserieslinechart.WithYLabelFormatter(yf))
}
if xf := s.XAxis.Format.labelFormatter(); xf != nil && s.XAxis.Format.Kind == "time" {
	// timeserieslinechart passes X label values in the same unit its default
	// DateTimeLabelFormatter receives. Read that function in
	// linechart/timeserieslinechart/timeserieslinechart.go and mirror its
	// value→time conversion (seconds vs milliseconds) inside a closure:
	layout := s.XAxis.Format.Layout
	if layout == "" {
		layout = "2006-01-02"
	}
	opts = append(opts, timeserieslinechart.WithXLabelFormatter(
		makeTimeAxisFormatter(layout))) // implement to match the default formatter's unit
}
```

Implement `makeTimeAxisFormatter(layout string) linechart.LabelFormatter` in build.go after reading `DateTimeLabelFormatter` — mirror its exact value→time conversion, changing only the layout. If the default formatter's unit is ambiguous, replicate its body verbatim with the layout swapped.

- [ ] **Step 5: Run the full spec suite**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -v && go vet ./spec/
```

Expected: PASS, existing goldens unchanged (formatters only activate when Formats are set).

- [ ] **Step 6: Commit**

```bash
cd /Users/evan/projects/ntcharts
git add spec/format.go spec/format_test.go spec/build.go
git commit -m "feat(spec): format engine (number/percent/currency/si/time) + timeseries wiring"
```

---

### Task 3: Build line charts (wavelinechart)

**Files:**
- Modify: `spec/build.go` (replace the `ChartTypeLine` not-implemented branch)
- Create: `spec/build_line_test.go`

**Interfaces:**
- Consumes: schema v1, `seriesStyle` (existing, build.go:212), `pointFloat` (existing helper), `Format.labelFormatter`.
- Produces: `buildLine(s Spec) (any, error)` returning `*wavelinechart.Model`; a shared helper `resolveXFloat(s Spec, p DataPoint, idx int) float64` reused by Task 4.

- [ ] **Step 1: Write the failing tests**

Create `spec/build_line_test.go`:

```go
package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart"
)

func lineSpec() Spec {
	return Spec{
		Type: ChartTypeLine, Width: 40, Height: 10,
		Data: Data{Series: []Series{
			{Name: "a", Color: "#ff0000", Values: []DataPoint{
				{X: 0.0, Y: 1}, {X: 1.0, Y: 3}, {X: 2.0, Y: 2}, {X: 3.0, Y: 5},
			}},
			{Name: "b", Values: []DataPoint{
				{X: 0.0, Y: 4}, {X: 1.0, Y: 2}, {X: 2.0, Y: 4}, {X: 3.0, Y: 1},
			}},
		}},
	}
}

func TestBuildLine(t *testing.T) {
	got, err := Build(lineSpec())
	if err != nil {
		t.Fatalf("Build(line): %v", err)
	}
	m, ok := got.(*wavelinechart.Model)
	if !ok {
		t.Fatalf("Build(line) returned %T, want *wavelinechart.Model", got)
	}
	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("line chart view is empty")
	}
}

func TestBuildLineIndexXWhenOmitted(t *testing.T) {
	s := lineSpec()
	for i := range s.Data.Series {
		for j := range s.Data.Series[i].Values {
			s.Data.Series[i].Values[j].X = nil // rely on index-as-X
		}
	}
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(line, index X): %v", err)
	}
}

func TestBuildLinePinnedYRange(t *testing.T) {
	s := lineSpec()
	s.YAxis.Min = f64(0)
	s.YAxis.Max = f64(10)
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(line, pinned Y): %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -run TestBuildLine -v
```

Expected: FAIL — `Build for chart type "line" is not yet implemented`.

- [ ] **Step 3: Implement `buildLine`**

In `spec/build.go`, route `case ChartTypeLine: return buildLine(s)` and add:

```go
// resolveXFloat resolves a point's numeric X: the point's own X, else the
// shared Data.XAxisData at idx, else the index itself.
func resolveXFloat(s Spec, p DataPoint, idx int) float64 {
	if p.X != nil {
		if x, err := pointFloat(p.X); err == nil {
			return x
		}
	}
	if idx < len(s.Data.XAxisData) {
		if x, err := pointFloat(s.Data.XAxisData[idx]); err == nil {
			return x
		}
	}
	return float64(idx)
}

// buildLine renders a numeric-X line chart onto a wavelinechart model.
func buildLine(s Spec) (any, error) {
	var opts []wavelinechart.Option
	if s.YAxis.Min != nil && s.YAxis.Max != nil {
		opts = append(opts, wavelinechart.WithYRange(*s.YAxis.Min, *s.YAxis.Max))
	}
	m := wavelinechart.New(s.Width, s.Height, opts...)
	for i, ser := range s.Data.Series {
		name := ser.Name
		m.SetDataSetStyles(name, runes.ArcLineStyle, seriesStyle(ser, i, s.Theme))
		for j, p := range ser.Values {
			m.PlotDataSet(name, canvas.Float64Point{X: resolveXFloat(s, p, j), Y: p.Y})
		}
	}
	m.DrawAll()
	return &m, nil
}
```

Add imports as needed: `github.com/NimbleMarkets/ntcharts/v2/canvas`, `github.com/NimbleMarkets/ntcharts/v2/canvas/runes`, `github.com/NimbleMarkets/ntcharts/v2/linechart/wavelinechart` (check existing import block — runes is already imported).

Note on label formatters: `wavelinechart` exposes no formatter options. If `wavelinechart.Model` embeds `linechart.Model` with exported formatter fields or setters (check `linechart/linechart.go` for `SetXLabelFormatter`/exported field), set them from `s.XAxis.Format.labelFormatter()`/`s.YAxis.Format.labelFormatter()` before `DrawAll()`; if no public hook exists, leave a `// TODO(spec-v2):` comment and note it in the README status table — do NOT reach into unexported state.

- [ ] **Step 4: Run tests to verify they pass, then capture a golden Example**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -run TestBuildLine -v
```

Expected: PASS. Then append to `spec/example_test.go` an `ExampleBuild_line` mirroring the existing `ExampleBuild_bar` pattern: build `lineSpec()`-shaped data inline, print `fmt.Println(m.View())`, run once to capture the actual rendered output, and paste it as the `// Output:` block. Verify:

```bash
go test ./spec/ -run ExampleBuild_line -v
```

Expected: PASS with the captured golden.

- [ ] **Step 5: Full suite + commit**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -v && go vet ./spec/
git add spec/build.go spec/build_line_test.go spec/example_test.go
git commit -m "feat(spec): Build() line charts via wavelinechart"
```

---

### Task 4: Build scatter charts (linechart primitives)

**Files:**
- Modify: `spec/build.go` (replace the `ChartTypeScatter` branch)
- Create: `spec/build_scatter_test.go`

**Interfaces:**
- Consumes: `resolveXFloat` (Task 3), `seriesStyle`, `Format.labelFormatter`, base `linechart` API (signatures in Global Constraints).
- Produces: `buildScatter(s Spec) (any, error)` returning `*linechart.Model`.

- [ ] **Step 1: Write the failing tests**

Create `spec/build_scatter_test.go`:

```go
package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/linechart"
)

func scatterSpec() Spec {
	return Spec{
		Type: ChartTypeScatter, Width: 40, Height: 12,
		XAxis: XAxis{Format: Format{Kind: "si"}},
		Data: Data{Series: []Series{
			{Name: "japan", Color: "#22aadd", Values: []DataPoint{
				{X: 2100.0, Y: 31.5}, {X: 1980.0, Y: 33.1},
			}},
			{Name: "usa", Color: "#dd8822", Values: []DataPoint{
				{X: 2875.0, Y: 24.0}, {X: 3200.0, Y: 19.2}, {X: 3600.0, Y: 16.5},
			}},
		}},
	}
}

func TestBuildScatter(t *testing.T) {
	got, err := Build(scatterSpec())
	if err != nil {
		t.Fatalf("Build(scatter): %v", err)
	}
	m, ok := got.(*linechart.Model)
	if !ok {
		t.Fatalf("Build(scatter) returned %T, want *linechart.Model", got)
	}
	view := m.View()
	if !strings.Contains(view, "•") {
		t.Fatalf("scatter view has no point markers:\n%s", view)
	}
}

func TestBuildScatterEmptySeriesErrors(t *testing.T) {
	s := scatterSpec()
	for i := range s.Data.Series {
		s.Data.Series[i].Values = nil
	}
	if _, err := Build(s); err == nil {
		t.Fatal("expected error for scatter with no points")
	}
}

func TestBuildScatterPinnedY(t *testing.T) {
	s := scatterSpec()
	s.YAxis.Min = f64(0)
	s.YAxis.Max = f64(40)
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(scatter, pinned Y): %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -run TestBuildScatter -v
```

Expected: FAIL — not-yet-implemented error.

- [ ] **Step 3: Implement `buildScatter`**

Route `case ChartTypeScatter: return buildScatter(s)` and add:

```go
// buildScatter renders points onto a base linechart canvas. ntcharts has no
// scatter model; the spec surface owns range computation and point drawing.
// DataPoint.Size is accepted in the schema but ignored by this surface.
func buildScatter(s Spec) (any, error) {
	type styledPoint struct {
		x, y  float64
		style lipgloss.Style
	}
	var pts []styledPoint
	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for i, ser := range s.Data.Series {
		st := seriesStyle(ser, i, s.Theme)
		for j, p := range ser.Values {
			x := resolveXFloat(s, p, j)
			pts = append(pts, styledPoint{x: x, y: p.Y, style: st})
			minX, maxX = math.Min(minX, x), math.Max(maxX, x)
			minY, maxY = math.Min(minY, p.Y), math.Max(maxY, p.Y)
		}
	}
	if len(pts) == 0 {
		return nil, fmt.Errorf("spec: scatter requires at least one data point")
	}
	if s.YAxis.Min != nil {
		minY = *s.YAxis.Min
	}
	if s.YAxis.Max != nil {
		maxY = *s.YAxis.Max
	}
	if minX == maxX {
		minX, maxX = minX-1, maxX+1
	}
	if minY == maxY {
		minY, maxY = minY-1, maxY+1
	}
	var opts []linechart.Option
	if xf := s.XAxis.Format.labelFormatter(); xf != nil {
		opts = append(opts, linechart.WithXLabelFormatter(xf))
	}
	if yf := s.YAxis.Format.labelFormatter(); yf != nil {
		opts = append(opts, linechart.WithYLabelFormatter(yf))
	}
	m := linechart.New(s.Width, s.Height, minX, maxX, minY, maxY, opts...)
	m.DrawXYAxisAndLabel()
	for _, p := range pts {
		m.DrawRuneWithStyle(canvas.Float64Point{X: p.x, Y: p.y}, '•', p.style)
	}
	return &m, nil
}
```

Add `math` and `github.com/NimbleMarkets/ntcharts/v2/linechart` imports if missing.

- [ ] **Step 4: Run tests to verify they pass, capture golden Example**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -run TestBuildScatter -v
```

Expected: PASS. Append `ExampleBuild_scatter` to `spec/example_test.go` (same capture procedure as Task 3 Step 4), verify it passes.

- [ ] **Step 5: Full suite + commit**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -v && go vet ./spec/
git add spec/build.go spec/build_scatter_test.go spec/example_test.go
git commit -m "feat(spec): Build() scatter charts via linechart primitives"
```

---

### Task 5: Build heatmaps (heatmap package + gradient scale)

**Files:**
- Modify: `spec/build.go` (replace the `ChartTypeHeatmap` branch)
- Create: `spec/gradient.go`
- Create: `spec/build_heatmap_test.go`

**Interfaces:**
- Consumes: `HeatData`/`HeatCell`/`Theme.Gradient` (Task 1), heatmap API (Global Constraints).
- Produces: `buildHeatmap(s Spec) (any, error)` returning `*heatmap.Model`; `gradientScale(stops []string) ([]color.Color, error)` in gradient.go.

- [ ] **Step 1: Write the failing tests**

Create `spec/build_heatmap_test.go`:

```go
package spec

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/heatmap"
)

func heatSpec() Spec {
	return Spec{
		Type: ChartTypeHeatmap, Width: 30, Height: 10,
		Heat: &HeatData{Cells: []HeatCell{
			{X: 0, Y: 0, Z: 1}, {X: 1, Y: 0, Z: 5}, {X: 2, Y: 0, Z: 9},
			{X: 0, Y: 1, Z: 3}, {X: 1, Y: 1, Z: 7}, {X: 2, Y: 1, Z: 2},
		}},
		Theme: Theme{Gradient: []string{"#000044", "#ff4400"}},
	}
}

func TestBuildHeatmapCells(t *testing.T) {
	got, err := Build(heatSpec())
	if err != nil {
		t.Fatalf("Build(heatmap): %v", err)
	}
	m, ok := got.(*heatmap.Model)
	if !ok {
		t.Fatalf("Build(heatmap) returned %T, want *heatmap.Model", got)
	}
	if strings.TrimSpace(m.View()) == "" {
		t.Fatal("heatmap view is empty")
	}
}

func TestBuildHeatmapMatrix(t *testing.T) {
	s := heatSpec()
	s.Heat = &HeatData{Matrix: [][]float64{{1, 5, 9}, {3, 7, 2}}}
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(heatmap, matrix): %v", err)
	}
}

func TestBuildHeatmapPinnedValueRange(t *testing.T) {
	s := heatSpec()
	s.Heat.MinValue = f64(0)
	s.Heat.MaxValue = f64(10)
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(heatmap, pinned range): %v", err)
	}
}

func TestGradientScale(t *testing.T) {
	cs, err := gradientScale([]string{"#000000", "#ff0000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 32 {
		t.Fatalf("gradient scale has %d steps, want 32", len(cs))
	}
	if _, err := gradientScale([]string{"not-a-color"}); err == nil {
		t.Fatal("expected error for invalid hex stop")
	}
	cs, err = gradientScale(nil)
	if err != nil || cs != nil {
		t.Fatalf("nil stops should give (nil, nil), got (%v, %v)", cs, err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -run 'TestBuildHeatmap|TestGradientScale' -v
```

Expected: FAIL — not-yet-implemented / `undefined: gradientScale`.

- [ ] **Step 3: Implement `spec/gradient.go`**

```go
// ntcharts - Copyright (c) 2026 Neomantra Corp.

package spec

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// gradientSteps is the resolution of an interpolated color scale.
const gradientSteps = 32

// gradientScale linearly interpolates ordered "#rrggbb" stops into a
// gradientSteps-long color scale for heatmap rendering. nil/empty stops
// return (nil, nil) so callers fall back to the chart default.
func gradientScale(stops []string) ([]color.Color, error) {
	if len(stops) == 0 {
		return nil, nil
	}
	rgbs := make([][3]float64, len(stops))
	for i, s := range stops {
		r, g, b, err := parseHexColor(s)
		if err != nil {
			return nil, err
		}
		rgbs[i] = [3]float64{r, g, b}
	}
	if len(rgbs) == 1 {
		rgbs = append(rgbs, rgbs[0])
	}
	out := make([]color.Color, gradientSteps)
	segs := len(rgbs) - 1
	for i := range out {
		t := float64(i) / float64(gradientSteps-1) * float64(segs)
		seg := int(t)
		if seg >= segs {
			seg = segs - 1
		}
		frac := t - float64(seg)
		a, b := rgbs[seg], rgbs[seg+1]
		out[i] = color.RGBA{
			R: uint8(a[0] + (b[0]-a[0])*frac),
			G: uint8(a[1] + (b[1]-a[1])*frac),
			B: uint8(a[2] + (b[2]-a[2])*frac),
			A: 0xff,
		}
	}
	return out, nil
}

func parseHexColor(s string) (r, g, b float64, err error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) != 6 {
		return 0, 0, 0, fmt.Errorf("spec: gradient stop %q is not #rrggbb", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("spec: gradient stop %q is not #rrggbb", s)
	}
	return float64((v >> 16) & 0xff), float64((v >> 8) & 0xff), float64(v & 0xff), nil
}
```

- [ ] **Step 4: Implement `buildHeatmap`**

Route `case ChartTypeHeatmap: return buildHeatmap(s)` and add to build.go:

```go
// buildHeatmap renders Heat data onto a heatmap model. Theme.Gradient (hex
// stops) becomes an interpolated color scale; absent, the package default
// grayscale applies.
func buildHeatmap(s Spec) (any, error) {
	if s.Heat == nil || (len(s.Heat.Cells) == 0 && len(s.Heat.Matrix) == 0) {
		return nil, fmt.Errorf("spec: heatmap requires Heat data (cells or matrix)")
	}
	var opts []heatmap.Option
	cs, err := gradientScale(s.Theme.Gradient)
	if err != nil {
		return nil, err
	}
	if cs != nil {
		opts = append(opts, heatmap.WithColorScale(cs))
	}
	if s.Heat.MinValue != nil && s.Heat.MaxValue != nil {
		opts = append(opts, heatmap.WithValueRange(*s.Heat.MinValue, *s.Heat.MaxValue))
	} else {
		opts = append(opts, heatmap.WithAutoValueRange())
	}
	m := heatmap.New(s.Width, s.Height, opts...)
	if len(s.Heat.Matrix) > 0 {
		m.PushAllMatrixRow(s.Heat.Matrix)
	} else {
		for _, c := range s.Heat.Cells {
			m.Push(heatmap.NewHeatPoint(c.X, c.Y, c.Z))
		}
	}
	m.Draw()
	return &m, nil
}
```

Add the `github.com/NimbleMarkets/ntcharts/v2/heatmap` import.

- [ ] **Step 5: Run tests, capture golden Example, full suite, commit**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -run 'TestBuildHeatmap|TestGradientScale' -v
```

Expected: PASS. Append `ExampleBuild_heatmap` to `spec/example_test.go` (capture procedure as before), then:

```bash
go test ./spec/ -v && go vet ./spec/
git add spec/build.go spec/gradient.go spec/build_heatmap_test.go spec/example_test.go
git commit -m "feat(spec): Build() heatmaps with gradient color scales"
```

---

### Task 6: Bar orientation + stacked semantics, README status

**Files:**
- Modify: `spec/build.go` (buildBar)
- Create: `spec/build_bar_test.go`
- Modify: `spec/README.md`

**Interfaces:**
- Consumes: `Options.Orientation`/`Options.Stacked` (Task 1), `barchart.WithHorizontalBars()`.
- Produces: final v1 bar semantics — multi-series bars require `Stacked: true` (ntcharts barchart has no grouped/side-by-side mode); `Orientation: "horizontal"` supported.

- [ ] **Step 1: Write the failing tests**

Create `spec/build_bar_test.go`:

```go
package spec

import (
	"strings"
	"testing"
)

func barSpec(nSeries int) Spec {
	s := Spec{
		Type: ChartTypeBar, Width: 40, Height: 10,
		XAxis: XAxis{Labels: []string{"Q1", "Q2", "Q3"}},
	}
	names := []string{"rev", "cost"}
	for i := 0; i < nSeries; i++ {
		s.Data.Series = append(s.Data.Series, Series{
			Name:   names[i],
			Values: []DataPoint{{Y: 3}, {Y: 5}, {Y: 2}},
		})
	}
	return s
}

func TestBuildBarMultiSeriesRequiresStacked(t *testing.T) {
	s := barSpec(2) // Stacked defaults to false
	_, err := Build(s)
	if err == nil || !strings.Contains(err.Error(), "stacked") {
		t.Fatalf("expected grouped-bars error mentioning stacked, got %v", err)
	}
	s.Options.Stacked = true
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(bar, stacked): %v", err)
	}
}

func TestBuildBarSingleSeriesNeedsNoStacked(t *testing.T) {
	if _, err := Build(barSpec(1)); err != nil {
		t.Fatalf("Build(bar, single series): %v", err)
	}
}

func TestBuildBarHorizontal(t *testing.T) {
	s := barSpec(1)
	s.Options.Orientation = OrientationHorizontal
	if _, err := Build(s); err != nil {
		t.Fatalf("Build(bar, horizontal): %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -run TestBuildBar -v
```

Expected: `TestBuildBarMultiSeriesRequiresStacked` FAILS (multi-series currently silently stacks without the flag); horizontal test may also fail. Note: if the pre-existing `ExampleBuild_bar` uses multiple series without `Stacked: true`, it must be updated in Step 3 to set the flag (the golden output must stay identical — stacking behavior is unchanged, only the gate is new).

- [ ] **Step 3: Implement in `buildBar`**

At the top of `buildBar`, before assembling BarData:

```go
if len(s.Data.Series) > 1 && !s.Options.Stacked {
	return nil, fmt.Errorf("spec: terminal bar charts cannot render grouped bars; set options.stacked=true or use a single series")
}
```

Where options are assembled:

```go
if s.Options.Orientation == OrientationHorizontal {
	opts = append(opts, barchart.WithHorizontalBars())
}
```

Update `spec/example_test.go`'s bar example to set `Stacked: true` if it has multiple series (same golden output).

- [ ] **Step 4: Run full suite**

```bash
cd /Users/evan/projects/ntcharts
go test ./spec/ -v && go vet ./spec/
```

Expected: PASS.

- [ ] **Step 5: Update `spec/README.md`**

- Status table: `ChartTypeLine`, `ChartTypeScatter`, `ChartTypeHeatmap` → `full` for `Build()` (terminal); their `ToECharts()` cells stay `scaffold`. Bar row: note "stacked + horizontal; grouped not supported by terminal surface".
- Document schema v1 in the README's type overview: `XAxis`/`YAxis`, `Format` kinds with one-line semantics (percent multiplies by 100; time expects ms-since-epoch X values with a Go layout), `Heat` (cells/matrix), `Theme.Gradient`, `DataPoint.Size` (accepted, terminal ignores), `Series.OHLC` (schema reserved; rendering in a later phase).
- Update the Quick start example to the new field locations (compile-check it mentally against the final types).

- [ ] **Step 6: Commit**

```bash
cd /Users/evan/projects/ntcharts
git add spec/build.go spec/build_bar_test.go spec/example_test.go spec/README.md
git commit -m "feat(spec): bar orientation + stacked gate; README for spec v1"
```
