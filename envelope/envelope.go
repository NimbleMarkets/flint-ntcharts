// Package envelope decodes the ntcharts-spec compiler envelope
// {spec, warnings, size} (or the {error: {message}} failure shape) emitted by
// any flint compiler backend — the native embedded-wasm compile.Runner or a
// browser/js host talking to a different backend. It depends on stdlib and
// the ntcharts spec package only, so it stays linkable under
// GOOS=js/GOARCH=wasm without pulling in wazero or the embedded wasm binary.
package envelope

import (
	"encoding/json"
	"fmt"

	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

// Warning mirrors flint's ChartWarning wire shape.
type Warning struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Channel  string `json:"channel,omitempty"`
	Field    string `json:"field,omitempty"`
}

// Size mirrors the envelope's `size` field.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// wireEnvelope is the on-the-wire shape of a compiler envelope.
type wireEnvelope struct {
	Spec     *spec.Spec      `json:"spec"`
	ECharts  json.RawMessage `json:"echarts"` // set instead of Spec by the raster renderer
	Warnings []Warning       `json:"warnings"`
	Size     Size            `json:"size"`
	// Error is set instead of Spec/Warnings/Size when the wasm/Node compiler
	// hits a compile-time failure (e.g. an unsupported chart type): it still
	// writes valid JSON to stdout and exits 0, so this is discriminated by
	// the top-level key present in the raw JSON, not by process exit status.
	// A wasm TRAP (engine-level failure, e.g. OOM) is a separate failure
	// layer: it does not produce this envelope at all and instead surfaces
	// as a process-level error from the backend (e.g. compile.Runner.CompileRaw).
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Parse decodes a compiler envelope. A {"error":{"message"}} envelope returns
// a non-nil error carrying the message verbatim. Warnings is never nil on
// success. The returned spec has already been shaped by flint; callers
// typically pass it straight to ntcharts spec.Build.
func Parse(raw []byte) (spec.Spec, []Warning, error) {
	var env wireEnvelope
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
	if env.hasECharts() {
		return spec.Spec{}, nil, fmt.Errorf("compile: this is a raster envelope (an ECharts option, not a spec); use ParseResult")
	}
	if env.Warnings == nil {
		env.Warnings = []Warning{}
	}
	if env.Spec == nil {
		return spec.Spec{}, env.Warnings, nil
	}
	return *env.Spec, env.Warnings, nil
}

func (e *wireEnvelope) hasECharts() bool {
	return len(e.ECharts) > 0 && string(e.ECharts) != "null"
}

// Result is a decoded compiler envelope of either kind. Exactly one of Spec
// and ECharts is set: Spec for the text renderer (draw it with spec.Build),
// ECharts for the raster renderer (flint's ECharts option, drawn as an image
// by the raster package).
type Result struct {
	Spec     *spec.Spec
	ECharts  json.RawMessage
	Warnings []Warning // never nil
	Size     Size
}

// ParseResult decodes a compiler envelope of either kind. A
// {"error":{"message"}} envelope returns a non-nil error carrying the message
// verbatim, as Parse does.
func ParseResult(raw []byte) (Result, error) {
	var env wireEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return Result{}, fmt.Errorf("compile: bad envelope: %w (raw: %.200s)", err, raw)
	}
	if env.Error != nil {
		return Result{}, fmt.Errorf("flint compile: %s", env.Error.Message)
	}
	if env.Warnings == nil {
		env.Warnings = []Warning{}
	}
	res := Result{Warnings: env.Warnings, Size: env.Size}
	switch {
	case env.hasECharts():
		res.ECharts = env.ECharts
	case env.Spec != nil:
		res.Spec = env.Spec
	default:
		return Result{}, fmt.Errorf("compile: envelope has neither a spec nor an echarts option (raw: %.200s)", raw)
	}
	return res, nil
}
