// Package tui implements the flint-tui live chart window: a bubbletea model
// that renders flint inputs (via the embedded wasm compiler), compile
// envelopes, or raw ntcharts-spec documents, re-fitting on terminal resize.
package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/flint-ntcharts/raster"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

type docKind int

const (
	docUnknown  docKind = iota
	docFlint            // ChartAssemblyInput: has chart_spec
	docEnvelope         // compile envelope: has spec
	docSpec             // raw ntcharts-spec: has type
	docError            // upstream error envelope: has error
)

// sniff classifies a JSON document per the frozen sniffing rules.
func sniff(raw []byte) (docKind, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return docUnknown, fmt.Errorf("invalid JSON: %w", err)
	}
	switch {
	case probe["chart_spec"] != nil:
		return docFlint, nil
	case probe["spec"] != nil:
		return docEnvelope, nil
	case probe["type"] != nil:
		return docSpec, nil
	case probe["error"] != nil:
		return docError, nil
	}
	return docUnknown, fmt.Errorf("document is neither flint input (chart_spec), envelope (spec), ntcharts-spec (type), nor an error envelope (error)")
}

// renderedMsg carries an async render result back into Update. gen ties the
// result back to the generation of the model that requested it (see rerenderCmd
// in tui.go); Update drops stale results from superseded requests.
type renderedMsg struct {
	view     string
	img      image.Image // set instead of view by a raster render
	warnings []envelope.Warning
	err      error
	gen      int
}

type viewer interface{ View() string }

// renderDoc compiles/builds raw into a chart view sized w×h (the chart area,
// status line already excluded by the caller). Errors return err with view
// empty; the model decides what stays on screen.
func renderDoc(compiler Compiler, raw []byte, w, h int, opts FrameOptions) renderedMsg {
	kind, err := sniff(raw)
	if err != nil {
		return renderedMsg{err: err}
	}
	if kind == docError {
		var envErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &envErr); err != nil {
			return renderedMsg{err: fmt.Errorf("bad error envelope: %w", err)}
		}
		return renderedMsg{err: fmt.Errorf("%s", envErr.Error.Message)}
	}
	var s spec.Spec
	var warnings []envelope.Warning
	switch kind {
	case docFlint:
		if opts.Renderer != "" && opts.Renderer != "text" && opts.Renderer != "raster" {
			return renderedMsg{err: fmt.Errorf("unknown renderer %q: use \"text\" or \"raster\"", opts.Renderer)}
		}
		copts := []envelope.Option{envelope.WithBaseSize(w, h)}
		if wantsRaster(raw, opts.Renderer) {
			msg, reason := renderRaster(compiler, raw, w, h, opts.CellW, opts.CellH)
			if reason == nil {
				return msg
			}
			// Fall back to the text renderer, and say why.
			warnings = append(warnings, *reason)
			copts = append(copts, envelope.WithRenderer("text"))
			var terr error
			s, _, terr = compiler.Compile(context.Background(), raw, copts...)
			if terr != nil {
				return renderedMsg{err: fmt.Errorf("raster renderer: %s; the text renderer cannot draw it either (%w)", strings.TrimSuffix(reason.Message, "; showing the text chart"), terr)}
			}
			break
		}
		if opts.Renderer == "text" {
			// An explicit choice beats a "renderer" key in the document.
			copts = append(copts, envelope.WithRenderer("text"))
		}
		var cw []envelope.Warning
		s, cw, err = compiler.Compile(context.Background(), raw, copts...)
		if err != nil {
			if opts.Renderer == "text" {
				return renderedMsg{err: fmt.Errorf("the text renderer cannot draw this chart (%w); switch to the raster renderer", err)}
			}
			return renderedMsg{err: err}
		}
		warnings = cw
	case docEnvelope:
		var env struct {
			Spec     spec.Spec          `json:"spec"`
			Warnings []envelope.Warning `json:"warnings"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			return renderedMsg{err: fmt.Errorf("bad envelope: %w", err)}
		}
		s = env.Spec
		s.Width, s.Height = w, h
		warnings = env.Warnings
	case docSpec:
		if err := json.Unmarshal(raw, &s); err != nil {
			return renderedMsg{err: fmt.Errorf("bad ntcharts-spec: %w", err)}
		}
		s.Width, s.Height = w, h
	}
	if err := s.Validate(); err != nil {
		return renderedMsg{err: err}
	}
	model, err := spec.Build(s)
	if err != nil {
		return renderedMsg{err: err}
	}
	v, ok := model.(viewer)
	if !ok {
		return renderedMsg{err: fmt.Errorf("chart model %T has no View", model)}
	}
	return renderedMsg{view: v.View(), warnings: warnings}
}

// Render compiles and builds raw into a chart view sized to a w×h cell area,
// returning the rendered terminal string, any compiler warnings, and an error
// if the document could not be sniffed, compiled, or built. It is the same
// path the live Model uses, exposed for embedders such as an integrated editor.
//
// Render returns a string, so a document that asks for the raster renderer is
// drawn as text (with a raster-unavailable warning); use [RenderFrame] to get
// the image.
func Render(c Compiler, raw []byte, w, h int) (view string, warnings []envelope.Warning, err error) {
	m := renderDoc(c, raw, w, h, FrameOptions{})
	return m.view, m.warnings, m.err
}

// FrameOptions says how RenderFrame draws.
type FrameOptions struct {
	// CellW and CellH are the terminal cell size in pixels. 0 means "this host
	// cannot show images", and a raster document renders as text.
	CellW, CellH int
	// Renderer overrides the document's own "renderer": "text" or "raster".
	// Empty leaves the document in charge. A host's toggle key sets it.
	Renderer string
}

// RenderFrame is Render for hosts that can show images: a document with
// "renderer": "raster" (or opts.Renderer "raster") comes back as an image sized
// to the w×h cell area at opts.CellW×opts.CellH pixels per cell. When the raster
// renderer cannot draw the chart (go-analyze drew it blank or failed, or the
// compiler has no CompileResult) the frame is the text renderer's chart and
// carries a raster-fallback or raster-unavailable warning.
func RenderFrame(c Compiler, raw []byte, w, h int, opts FrameOptions) (Frame, error) {
	m := renderDoc(c, raw, w, h, opts)
	if m.err != nil {
		return Frame{}, m.err
	}
	return Frame{Text: m.view, Image: m.img, Warnings: m.warnings}, nil
}

// ToggleRenderer returns the renderer override that flips the chart on screen:
// "raster" if it is drawn as text now, "text" if as raster. current is the
// override in force ("" if none), and raw the document, whose own "renderer"
// decides when there is no override.
func ToggleRenderer(raw []byte, current string) string {
	if wantsRaster(raw, current) {
		return "text"
	}
	return "raster"
}

// wantsRaster reports whether the chart should be drawn by the raster renderer:
// the override if there is one, else the document's own "renderer".
func wantsRaster(raw []byte, override string) bool {
	switch override {
	case "raster":
		return true
	case "text":
		return false
	}
	var probe struct {
		Renderer string `json:"renderer"`
	}
	return json.Unmarshal(raw, &probe) == nil && probe.Renderer == "raster"
}

// ResultCompiler is a Compiler that can also return the raster renderer's
// ECharts option. compile.Runner implements it; a host adapter that only
// implements Compiler gets text charts.
type ResultCompiler interface {
	Compiler
	CompileResult(ctx context.Context, input []byte, opts ...envelope.Option) (envelope.Result, error)
}

// renderRaster compiles and draws a raster document. On success it returns
// the image frame and a nil warning; otherwise the warning says why the
// caller should fall back to text.
func renderRaster(c Compiler, raw []byte, w, h, cellW, cellH int) (renderedMsg, *envelope.Warning) {
	fallback := func(code, why string) (renderedMsg, *envelope.Warning) {
		return renderedMsg{}, &envelope.Warning{Severity: "warning", Code: code, Message: why}
	}
	rc, ok := c.(ResultCompiler)
	if !ok {
		return fallback("raster-unavailable", "this host's compiler cannot produce raster charts; showing the text chart")
	}
	if cellW <= 0 || cellH <= 0 {
		return fallback("raster-unavailable", "this host cannot show images; showing the text chart")
	}
	// flint's ECharts layout is in pixels, unlike the text renderer's cells.
	res, err := rc.CompileResult(context.Background(), raw, envelope.WithBaseSize(w*cellW, h*cellH), envelope.WithRenderer("raster"))
	if err != nil {
		return fallback("raster-fallback", "raster compile failed ("+err.Error()+"); showing the text chart")
	}
	if res.ECharts == nil {
		return fallback("raster-fallback", "the compiler returned no raster chart; showing the text chart")
	}
	img, err := raster.Render(res.ECharts, w*cellW, h*cellH)
	switch {
	case errors.Is(err, raster.ErrBlank):
		return fallback("raster-fallback", "raster rendered the chart blank; showing the text chart")
	case err != nil:
		return fallback("raster-fallback", "raster render failed ("+err.Error()+"); showing the text chart")
	}
	return renderedMsg{img: img, warnings: res.Warnings}, nil
}
