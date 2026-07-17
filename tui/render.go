// Package tui implements the flint-tui live chart window: a bubbletea model
// that renders flint inputs (via the embedded wasm compiler), compile
// envelopes, or raw ntcharts-spec documents, re-fitting on terminal resize.
package tui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/NimbleMarkets/flint-ntcharts/envelope"
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
	warnings []envelope.Warning
	err      error
	gen      int
}

type viewer interface{ View() string }

// renderDoc compiles/builds raw into a chart view sized w×h (the chart area,
// status line already excluded by the caller). Errors return err with view
// empty; the model decides what stays on screen.
func renderDoc(compiler Compiler, raw []byte, w, h int) renderedMsg {
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
		s, warnings, err = compiler.Compile(context.Background(), raw, envelope.WithBaseSize(w, h))
		if err != nil {
			return renderedMsg{err: err}
		}
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
