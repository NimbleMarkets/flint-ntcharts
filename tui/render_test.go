package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
)

func newTestRunner(t *testing.T) *compile.Runner {
	t.Helper()
	r, err := compile.New(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close(context.Background()) })
	return r
}

const flintDoc = `{
  "data": {"values": [
    {"product": "A", "revenue": 10}, {"product": "B", "revenue": 25}, {"product": "C", "revenue": 15}
  ]},
  "chart_spec": {"chartType": "Bar Chart",
    "encodings": {"x": {"field": "product"}, "y": {"field": "revenue"}}}
}`

const specDoc = `{
  "type": "bar", "width": 30, "height": 8,
  "x_axis": {"labels": ["Q1", "Q2"]},
  "data": {"series": [{"name": "rev", "values": [{"y": 3}, {"y": 5}]}]}
}`

func TestSniff(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want docKind
	}{
		{"flint", flintDoc, docFlint},
		{"direct spec", specDoc, docSpec},
		{"envelope", `{"spec": {"type":"bar","width":10,"height":5,"data":{"series":[{"name":"a","values":[{"y":1}]}]}}, "warnings": [], "size": {"width":10,"height":5}}`, docEnvelope},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := sniff([]byte(c.doc))
			if err != nil || got != c.want {
				t.Fatalf("sniff = %v, %v; want %v", got, err, c.want)
			}
		})
	}
	if _, err := sniff([]byte(`{"foo": 1}`)); err == nil {
		t.Fatal("expected sniff error for unknown document shape")
	}
	if _, err := sniff([]byte(`not json`)); err == nil {
		t.Fatal("expected sniff error for invalid JSON")
	}
}

func TestRenderDocFlint(t *testing.T) {
	r := newTestRunner(t)
	msg := renderDoc(r, []byte(flintDoc), 60, 20)
	if msg.err != nil {
		t.Fatalf("renderDoc(flint): %v", msg.err)
	}
	if strings.TrimSpace(msg.view) == "" {
		t.Fatal("empty chart view")
	}
}

func TestRenderDocDirectSpecFitsWindow(t *testing.T) {
	r := newTestRunner(t)
	msg := renderDoc(r, []byte(specDoc), 44, 15)
	if msg.err != nil {
		t.Fatalf("renderDoc(spec): %v", msg.err)
	}
	lines := strings.Split(strings.TrimRight(msg.view, "\n"), "\n")
	if len(lines) > 15 { // renderDoc receives the already-reduced chart-area height (15 here) and reserves nothing itself
		t.Fatalf("direct spec not fitted: %d lines > 15", len(lines))
	}
}

func TestRenderDocBadDocKeepsError(t *testing.T) {
	r := newTestRunner(t)
	msg := renderDoc(r, []byte(`{"chart_spec": {"chartType": "Rose Chart", "encodings": {}}, "data": {"values": []}}`), 40, 12)
	if msg.err == nil {
		t.Fatal("expected compile error to surface")
	}
	if !strings.Contains(msg.err.Error(), "Unknown chart type") {
		t.Fatalf("error should carry the compiler message verbatim, got: %v", msg.err)
	}
}
