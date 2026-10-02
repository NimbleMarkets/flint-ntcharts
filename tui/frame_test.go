package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/ntcharts/v2/spec"
)

const groupedDoc = `{
  "renderer": "raster",
  "data": {"values": [
    {"c":"Laptop","n":3,"g":"USA"},{"c":"Phone","n":5,"g":"USA"},{"c":"Tablet","n":1,"g":"USA"},
    {"c":"Laptop","n":2,"g":"China"},{"c":"Phone","n":4,"g":"China"},{"c":"Tablet","n":2,"g":"China"}]},
  "chart_spec": {"chartType":"Grouped Bar Chart",
    "encodings": {"x":{"field":"c"},"y":{"field":"n"},"group":{"field":"g"}}}
}`

// raster draws nothing for a heatmap, which the text renderer does draw.
const heatmapRasterDoc = `{
  "renderer": "raster",
  "data": {"values": [
    {"day":"Mon","when":"AM","v":3},{"day":"Mon","when":"PM","v":9},
    {"day":"Tue","when":"AM","v":5},{"day":"Tue","when":"PM","v":1}]},
  "chart_spec": {"chartType":"Heatmap",
    "encodings": {"x":{"field":"when"},"y":{"field":"day"},"color":{"field":"v"}}}
}`

// the text renderer cannot draw a waterfall and raster draws it blank
const waterfallRasterDoc = `{
  "renderer": "raster",
  "data": {"values": [{"s":"Rev","a":100},{"s":"Cost","a":-40},{"s":"Tax","a":-10},{"s":"Net","a":50}]},
  "chart_spec": {"chartType":"Waterfall Chart","encodings": {"x":{"field":"s"},"y":{"field":"a"}}}
}`

const barRasterDoc = `{
  "renderer": "raster",
  "data": {"values": [{"p":"A","r":10},{"p":"B","r":25},{"p":"C","r":15}]},
  "chart_spec": {"chartType":"Bar Chart","encodings": {"x":{"field":"p"},"y":{"field":"r"}}}
}`

func warnCodes(ws []envelope.Warning) []string {
	var out []string
	for _, w := range ws {
		out = append(out, w.Code)
	}
	return out
}

func hasCode(ws []envelope.Warning, code string) bool {
	for _, w := range ws {
		if w.Code == code {
			return true
		}
	}
	return false
}

func TestRenderFrameRasterDrawsAnImage(t *testing.T) {
	r := newTestRunner(t)
	f, err := RenderFrame(r, []byte(groupedDoc), 40, 12, FrameOptions{CellW: 8, CellH: 16})
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	if f.Image == nil || f.Text != "" {
		t.Fatalf("want an image frame, got text=%q image=%v", f.Text, f.Image)
	}
	if b := f.Image.Bounds(); b.Dx() != 40*8 || b.Dy() != 12*16 {
		t.Fatalf("image is %dx%d px, want the 40x12 cell area at 8x16 = 320x192", b.Dx(), b.Dy())
	}
	if len(f.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnCodes(f.Warnings))
	}
}

// flint's ECharts layout works in pixels. Handing it the chart area in cells
// (40x12) makes it think the canvas is a few dozen pixels wide and drop
// categories, with an overflow warning, so the raster compile gets pixels.
func TestRenderFrameRasterLaysOutInPixelsNotCells(t *testing.T) {
	r := newTestRunner(t)
	f, err := RenderFrame(r, []byte(groupedDoc), 40, 12, FrameOptions{CellW: 8, CellH: 16})
	if err != nil {
		t.Fatal(err)
	}
	if hasCode(f.Warnings, "overflow") {
		t.Fatalf("categories were dropped as if the canvas were 40x12 px: %v", f.Warnings)
	}
}

func TestRenderFrameBlankRasterFallsBackToText(t *testing.T) {
	r := newTestRunner(t)
	f, err := RenderFrame(r, []byte(heatmapRasterDoc), 40, 12, FrameOptions{CellW: 8, CellH: 16})
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	if f.Image != nil || strings.TrimSpace(f.Text) == "" {
		t.Fatalf("want the text chart, got text=%q image=%v", f.Text, f.Image)
	}
	if !hasCode(f.Warnings, "raster-fallback") {
		t.Fatalf("warnings %v lack raster-fallback", warnCodes(f.Warnings))
	}
}

func TestRenderFrameNoTextFallbackIsAnError(t *testing.T) {
	r := newTestRunner(t)
	_, err := RenderFrame(r, []byte(waterfallRasterDoc), 40, 12, FrameOptions{CellW: 8, CellH: 16})
	if err == nil {
		t.Fatal("want an error: raster drew nothing and text cannot draw a waterfall")
	}
	for _, want := range []string{"raster", "blank", "text renderer"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err, want)
		}
	}
}

// textOnly hides a compiler's CompileResult, like a host adapter that only
// implements Compiler.
type textOnly struct{ c Compiler }

func (t textOnly) Compile(ctx context.Context, in []byte, opts ...envelope.Option) (spec.Spec, []envelope.Warning, error) {
	return t.c.Compile(ctx, in, opts...)
}

func TestRenderFrameRasterNeedsAResultCompiler(t *testing.T) {
	r := newTestRunner(t)
	f, err := RenderFrame(textOnly{r}, []byte(barRasterDoc), 40, 12, FrameOptions{CellW: 8, CellH: 16})
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	if f.Image != nil || strings.TrimSpace(f.Text) == "" {
		t.Fatalf("want the text chart, got text=%q image=%v", f.Text, f.Image)
	}
	if !hasCode(f.Warnings, "raster-unavailable") {
		t.Fatalf("warnings %v lack raster-unavailable", warnCodes(f.Warnings))
	}
}

func TestRenderFrameWithoutCellPixelSizeIsText(t *testing.T) {
	r := newTestRunner(t)
	f, err := RenderFrame(r, []byte(barRasterDoc), 40, 12, FrameOptions{CellW: 0, CellH: 0})
	if err != nil {
		t.Fatalf("RenderFrame: %v", err)
	}
	if f.Image != nil || strings.TrimSpace(f.Text) == "" || !hasCode(f.Warnings, "raster-unavailable") {
		t.Fatalf("want text + raster-unavailable, got text=%q image=%v warnings=%v", f.Text, f.Image, warnCodes(f.Warnings))
	}
}

// Render predates raster and returns a string, so a raster document renders
// as text through it.
func TestRenderStringAPIRendersRasterDocumentsAsText(t *testing.T) {
	r := newTestRunner(t)
	view, warns, err := Render(r, []byte(barRasterDoc), 40, 12)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.TrimSpace(view) == "" || !hasCode(warns, "raster-unavailable") {
		t.Fatalf("view=%q warnings=%v", view, warnCodes(warns))
	}
}

func TestRenderFrameTextDocumentsAreUnchanged(t *testing.T) {
	r := newTestRunner(t)
	f, err := RenderFrame(r, []byte(flintDoc), 40, 12, FrameOptions{CellW: 8, CellH: 16})
	if err != nil {
		t.Fatal(err)
	}
	if f.Image != nil || strings.TrimSpace(f.Text) == "" || len(f.Warnings) != 0 {
		t.Fatalf("text=%q image=%v warnings=%v", f.Text, f.Image, warnCodes(f.Warnings))
	}
}

const barDoc = `{
  "data": {"values": [{"p":"A","r":10},{"p":"B","r":25},{"p":"C","r":15}]},
  "chart_spec": {"chartType":"Bar Chart","encodings": {"x":{"field":"p"},"y":{"field":"r"}}}
}`

var cells = FrameOptions{CellW: 8, CellH: 16}

func withRenderer(r string) FrameOptions { o := cells; o.Renderer = r; return o }

func TestRendererOverrideDrawsATextDocumentAsRaster(t *testing.T) {
	r := newTestRunner(t)
	f, err := RenderFrame(r, []byte(barDoc), 40, 12, withRenderer("raster"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Image == nil || f.Text != "" {
		t.Fatalf("want an image, got text=%q image=%v", f.Text, f.Image)
	}
}

func TestRendererOverrideDrawsARasterDocumentAsText(t *testing.T) {
	r := newTestRunner(t)
	f, err := RenderFrame(r, []byte(barRasterDoc), 40, 12, withRenderer("text"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Image != nil || strings.TrimSpace(f.Text) == "" || len(f.Warnings) != 0 {
		t.Fatalf("want a clean text chart, got text=%q image=%v warnings=%v", f.Text, f.Image, warnCodes(f.Warnings))
	}
}

func TestRendererOverrideToTextExplainsWhenTextCannotDrawIt(t *testing.T) {
	r := newTestRunner(t)
	_, err := RenderFrame(r, []byte(groupedDoc), 40, 12, withRenderer("text"))
	if err == nil {
		t.Fatal("want an error: the text renderer cannot draw grouped bars")
	}
	for _, want := range []string{"text renderer", "cannot", "raster"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q lacks %q", err, want)
		}
	}
}

func TestRendererOverrideToRasterFallsBackWhenRasterCannotDrawIt(t *testing.T) {
	r := newTestRunner(t)
	heatmap := strings.Replace(heatmapRasterDoc, `"renderer": "raster",`, "", 1)
	f, err := RenderFrame(r, []byte(heatmap), 40, 12, withRenderer("raster"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Image != nil || !hasCode(f.Warnings, "raster-fallback") {
		t.Fatalf("want the text heatmap with raster-fallback, got image=%v warnings=%v", f.Image, warnCodes(f.Warnings))
	}
}

func TestRendererOverrideRejectsUnknownNames(t *testing.T) {
	r := newTestRunner(t)
	if _, err := RenderFrame(r, []byte(barDoc), 40, 12, withRenderer("pixels")); err == nil {
		t.Fatal("an unknown renderer must be an error")
	}
}

func TestToggleRenderer(t *testing.T) {
	cases := []struct{ doc, override, want string }{
		{barDoc, "", "raster"},     // a text document flips to raster
		{barRasterDoc, "", "text"}, // a raster document flips to text
		{barDoc, "raster", "text"}, // an override flips back
		{barRasterDoc, "text", "raster"},
	}
	for _, c := range cases {
		if got := ToggleRenderer([]byte(c.doc), c.override); got != c.want {
			t.Errorf("ToggleRenderer(doc, %q) = %q, want %q", c.override, got, c.want)
		}
	}
}
