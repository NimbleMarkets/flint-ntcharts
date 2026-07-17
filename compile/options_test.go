package compile

import (
	"encoding/json"
	"testing"

	"github.com/NimbleMarkets/flint-ntcharts/envelope"
)

// TestOptionAliasesUsable confirms compile.Option/WithBaseSize/WithCanvasSize
// remain usable as thin aliases of the envelope package after the option
// logic itself moved to envelope/options.go (see envelope/options_test.go
// for the moved logic tests).
func TestOptionAliasesUsable(t *testing.T) {
	in := []byte(`{"data":{"values":[]},"chart_spec":{"chartType":"Bar Chart","encodings":{}}}`)
	var opts []Option
	opts = append(opts, WithBaseSize(48, 16))
	opts = append(opts, WithCanvasSize(100, 40))
	out, err := envelope.Apply(in, opts)
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
		t.Fatalf("baseSize not injected via aliased option: %v", bs)
	}
	csz := cs["canvasSize"].(map[string]any)
	if csz["width"].(float64) != 100 {
		t.Fatalf("canvasSize not injected via aliased option: %v", csz)
	}
}
