package envelope

import (
	"encoding/json"
	"strings"
	"testing"
)

const textEnvelope = `{"spec":{"type":"bar","width":10,"height":5,"data":{"series":[{"name":"a","values":[{"y":1}]}]}},"warnings":[{"severity":"info","code":"x","message":"m"}],"size":{"width":10,"height":5}}`
const rasterEnvelope = `{"echarts":{"series":[{"type":"bar","data":[1,2]}]},"warnings":[],"size":{"width":80,"height":24}}`

func TestParseResultText(t *testing.T) {
	r, err := ParseResult([]byte(textEnvelope))
	if err != nil {
		t.Fatal(err)
	}
	if r.Spec == nil || r.Spec.Type != "bar" || r.ECharts != nil {
		t.Fatalf("want a spec result, got %+v", r)
	}
	if len(r.Warnings) != 1 || r.Size.Width != 10 {
		t.Fatalf("warnings/size lost: %+v", r)
	}
}

func TestParseResultRaster(t *testing.T) {
	r, err := ParseResult([]byte(rasterEnvelope))
	if err != nil {
		t.Fatal(err)
	}
	if r.Spec != nil || r.ECharts == nil {
		t.Fatalf("want an echarts result, got %+v", r)
	}
	var opt struct{ Series []struct{ Type string } }
	if err := json.Unmarshal(r.ECharts, &opt); err != nil || len(opt.Series) != 1 || opt.Series[0].Type != "bar" {
		t.Fatalf("echarts option mangled: %s (%v)", r.ECharts, err)
	}
	if r.Warnings == nil || r.Size.Width != 80 || r.Size.Height != 24 {
		t.Fatalf("warnings/size wrong: %+v", r)
	}
}

func TestParseResultError(t *testing.T) {
	_, err := ParseResult([]byte(`{"error":{"message":"boom"}}`))
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseResultGarbage(t *testing.T) {
	if _, err := ParseResult([]byte(`nope`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseResultNeitherSpecNorOption(t *testing.T) {
	if _, err := ParseResult([]byte(`{"warnings":[],"size":{"width":1,"height":1}}`)); err == nil {
		t.Fatal("an envelope with no spec and no echarts must be an error")
	}
}

// Parse predates the raster renderer and returns only a spec. A raster
// envelope must not decode to an empty spec and look like success.
func TestParseRejectsRasterEnvelope(t *testing.T) {
	_, _, err := Parse([]byte(rasterEnvelope))
	if err == nil || !strings.Contains(err.Error(), "ParseResult") {
		t.Fatalf("err = %v, want one pointing at ParseResult", err)
	}
}

func TestWithRenderer(t *testing.T) {
	out, err := Apply([]byte(`{"chart_spec":{}}`), []Option{WithRenderer("raster")})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	json.Unmarshal(out, &doc)
	if doc["renderer"] != "raster" {
		t.Fatalf("renderer = %v", doc["renderer"])
	}
	if _, err := Apply([]byte(`{}`), []Option{WithRenderer("pixels")}); err == nil {
		t.Fatal("unknown renderer must be an error")
	}
}
