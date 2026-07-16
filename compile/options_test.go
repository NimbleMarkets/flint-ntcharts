package compile

import (
	"encoding/json"
	"testing"
)

func TestWithBaseSizeInjection(t *testing.T) {
	in := []byte(`{"data":{"values":[]},"chart_spec":{"chartType":"Bar Chart","encodings":{}}}`)
	out, err := applyOptions(in, []Option{WithBaseSize(48, 16)})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	cs := doc["chart_spec"].(map[string]any)
	bs := cs["baseSize"].(map[string]any)
	if bs["width"].(float64) != 48 || bs["height"].(float64) != 16 {
		t.Fatalf("baseSize not injected: %v", bs)
	}
}

func TestWithCanvasSizeOverridesExisting(t *testing.T) {
	in := []byte(`{"data":{"values":[]},"chart_spec":{"chartType":"Bar Chart","encodings":{},"canvasSize":{"width":1,"height":1}}}`)
	out, err := applyOptions(in, []Option{WithCanvasSize(100, 40)})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	cs := doc["chart_spec"].(map[string]any)["canvasSize"].(map[string]any)
	if cs["width"].(float64) != 100 {
		t.Fatalf("canvasSize not overridden: %v", cs)
	}
}

func TestNoOptionsPassthroughBytes(t *testing.T) {
	in := []byte(`{"x": 1}`)
	out, err := applyOptions(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Fatal("no-option path must not re-serialize the input")
	}
}
