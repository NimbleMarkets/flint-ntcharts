// Package tui implements the flint-tui live chart window: a bubbletea model
// that renders flint inputs (via the embedded wasm compiler), compile
// envelopes, or raw ntcharts-spec documents, re-fitting on terminal resize.
package tui

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

type docKind int

const (
	docUnknown  docKind = iota
	docFlint            // ChartAssemblyInput: has chart_spec
	docEnvelope         // compile envelope: has spec
	docSpec             // raw ntcharts-spec: has type
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
	}
	return docUnknown, fmt.Errorf("document is neither flint input (chart_spec), envelope (spec), nor ntcharts-spec (type)")
}

// renderedMsg carries an async render result back into Update.
type renderedMsg struct {
	view     string
	warnings []compile.Warning
	err      error
}

type viewer interface{ View() string }

// renderDoc compiles/builds raw into a chart view sized w×h (the chart area,
// status line already excluded by the caller). Errors return err with view
// empty; the model decides what stays on screen.
func renderDoc(runner *compile.Runner, raw []byte, w, h int) renderedMsg {
	kind, err := sniff(raw)
	if err != nil {
		return renderedMsg{err: err}
	}
	var s spec.Spec
	var warnings []compile.Warning
	switch kind {
	case docFlint:
		s, warnings, err = runner.Compile(context.Background(), raw, compile.WithBaseSize(w, h))
		if err != nil {
			return renderedMsg{err: err}
		}
	case docEnvelope:
		var env struct {
			Spec spec.Spec `json:"spec"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			return renderedMsg{err: fmt.Errorf("bad envelope: %w", err)}
		}
		s = env.Spec
		s.Width, s.Height = w, h
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
