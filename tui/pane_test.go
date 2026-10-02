package tui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

func solid(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestPaneShowsText(t *testing.T) {
	p := NewPane()
	p.SetSize(20, 6)
	p.Apply(Frame{Text: "hello\nchart"})
	if got := p.View(); got != "hello\nchart" {
		t.Fatalf("View = %q", got)
	}
}

func TestPaneShowsAnImageInTheCellArea(t *testing.T) {
	p := NewPane()
	p.SetSize(20, 6)
	p.Apply(Frame{Image: solid(160, 96, color.RGBA{R: 200, G: 40, B: 40, A: 255})})
	v := p.View()
	if lines := strings.Split(v, "\n"); len(lines) != 6 {
		t.Fatalf("image is %d lines, want the 6 rows of the pane", len(lines))
	}
	if strings.TrimSpace(v) == "" {
		t.Fatal("image drew nothing")
	}
}

func TestPaneTextReplacesAnImage(t *testing.T) {
	p := NewPane()
	p.SetSize(20, 6)
	p.Apply(Frame{Image: solid(160, 96, color.White)})
	p.Apply(Frame{Text: "back to text"})
	if got := p.View(); got != "back to text" {
		t.Fatalf("View = %q", got)
	}
}

func TestPaneReportsItsCellPixelSize(t *testing.T) {
	p := NewPane()
	w, h := p.CellPixelSize()
	if w <= 0 || h <= 0 {
		t.Fatalf("cell pixel size %dx%d", w, h)
	}
}

func TestPaneStartsInGlyphMode(t *testing.T) {
	p := NewPane()
	if p.Mode() != picture.PictureGlyph {
		t.Fatal("a pane must work without Kitty until the terminal says it can")
	}
}

// forceKitty sets the process-wide capability for one test.
func forceKitty(t *testing.T, c picture.KittyCapability) {
	t.Helper()
	prev := picture.KittySupported()
	picture.ForceKittyCapability(c)
	t.Cleanup(func() { picture.ForceKittyCapability(prev) })
}

func TestPaneToggleSaysWhatItDid(t *testing.T) {
	forceKitty(t, picture.KittyCapabilitySupported)
	p := NewPane()
	p.SetSize(20, 6)
	p.Apply(Frame{Image: solid(160, 96, color.White)})

	_, note := p.Toggle()
	if p.Mode() != picture.PictureKitty || !strings.Contains(note, "Kitty graphics") {
		t.Fatalf("mode=%v note=%q, want Kitty", p.Mode(), note)
	}
	_, note = p.Toggle()
	if p.Mode() != picture.PictureGlyph || !strings.Contains(note, "glyphs") {
		t.Fatalf("mode=%v note=%q, want glyphs", p.Mode(), note)
	}
}

// The picture model refuses Kitty mode unless the terminal said it supports it.
// A silent no-op looks like a broken key, so the pane explains and names the
// override.
func TestPaneToggleExplainsWhyItCannotEnterKitty(t *testing.T) {
	forceKitty(t, picture.KittyCapabilityUnsupported)
	p := NewPane()
	p.SetSize(20, 6)
	p.Apply(Frame{Image: solid(160, 96, color.White)})

	_, note := p.Toggle()
	if p.Mode() != picture.PictureGlyph {
		t.Fatal("must stay in glyph mode")
	}
	for _, want := range []string{"did not report", "FLINT_KITTY=1"} {
		if !strings.Contains(note, want) {
			t.Fatalf("note %q lacks %q", note, want)
		}
	}
}

func TestPaneToggleWithATextChartSaysSo(t *testing.T) {
	forceKitty(t, picture.KittyCapabilitySupported)
	p := NewPane()
	p.Apply(Frame{Text: "text chart"})
	_, note := p.Toggle()
	if !strings.Contains(note, "text") {
		t.Fatalf("note %q should say the chart is text", note)
	}
}

func TestFlintKittyForcesTheCapability(t *testing.T) {
	forceKitty(t, picture.KittyCapabilityUnsupported)
	t.Setenv("FLINT_KITTY", "1")
	NewPane()
	if picture.KittySupported() != picture.KittyCapabilitySupported {
		t.Fatal("FLINT_KITTY=1 must force Kitty on")
	}
	t.Setenv("FLINT_KITTY", "0")
	NewPane()
	if picture.KittySupported() != picture.KittyCapabilityUnsupported {
		t.Fatal("FLINT_KITTY=0 must force Kitty off")
	}
}

func TestPaneStaysInGlyphModeWhenForcedOff(t *testing.T) {
	forceKitty(t, picture.KittyCapabilityUnknown)
	t.Setenv("FLINT_KITTY", "0")
	p := NewPane()
	p.Apply(Frame{Image: solid(160, 96, color.White)})
	p.Update(nil)
	if p.Mode() != picture.PictureGlyph {
		t.Fatal("FLINT_KITTY=0 must keep glyph mode")
	}
}
