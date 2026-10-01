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
