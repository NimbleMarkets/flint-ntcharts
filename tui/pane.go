package tui

import (
	"image"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/NimbleMarkets/flint-ntcharts/envelope"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

// Frame is one rendered chart: terminal text from the text renderer, or an
// image from the raster renderer for a [Pane] to place. Exactly one of Text
// and Image is set.
type Frame struct {
	Text     string
	Image    image.Image
	Warnings []envelope.Warning
}

// Pane shows the chart area of a viewer. It holds the latest [Frame] and draws
// either its text or its image, so a host (flint-tui, flint-edit) treats both
// renderers alike: send it the frames, forward it the messages, and put View
// where the chart goes.
//
// Images go through ntcharts' picture.Model: Kitty graphics when the terminal
// answers the capability probe, half-block glyphs otherwise. Kitty output is a
// grid of placeholder cells, so it composes with surrounding text (a pane
// joined next to an editor, say) like any other string.
//
// A Pane is a value, like the models that hold it: keep the one Apply and
// Update return changes to.
type Pane struct {
	pic       picture.Model
	text      string
	hasImage  bool
	modeFixed bool // the user chose a mode; stop following the capability probe
}

// NewPane returns an empty pane in glyph mode.
//
// The Kitty probe only runs in terminals that look Kitty-aware (kitty,
// Ghostty, WezTerm, iTerm, tmux passthrough), so detection can miss. The
// FLINT_KITTY environment variable overrides it: 1 forces Kitty graphics on,
// 0 forces them off (glyphs only).
func NewPane() Pane {
	switch os.Getenv("FLINT_KITTY") {
	case "1":
		picture.ForceKittyCapability(picture.KittyCapabilitySupported)
	case "0":
		picture.ForceKittyCapability(picture.KittyCapabilityUnsupported)
	}
	return Pane{pic: picture.New()}
}

// Init starts the terminal probes (Kitty support, cell pixel size). Batch it
// with the host's own Init.
func (p *Pane) Init() tea.Cmd { return p.pic.Init() }

// CellPixelSize is the terminal cell size in pixels, used to size raster
// renders to the chart area.
func (p *Pane) CellPixelSize() (w, h int) { return p.pic.CellPixelSize() }

// Mode reports whether images are drawn with Kitty graphics or glyphs.
func (p *Pane) Mode() picture.PictureMode { return p.pic.Mode() }

// SetSize sets the chart area in terminal cells.
func (p *Pane) SetSize(cols, rows int) tea.Cmd { return p.pic.SetSize(cols, rows) }

// Apply makes f the pane's content. A text frame clears any image.
func (p *Pane) Apply(f Frame) tea.Cmd {
	if f.Image != nil {
		p.hasImage = true
		return p.pic.SetImage(f.Image)
	}
	p.text = f.Text
	if p.hasImage {
		p.hasImage = false
		return p.pic.SetImage(nil)
	}
	return nil
}

// Toggle switches an image between Kitty graphics and glyphs and returns a
// one-line note saying what happened, for a status line. It can decline: the
// picture model enters Kitty mode only when the terminal said it supports it,
// and the note then names the FLINT_KITTY override. After a toggle the pane
// stops switching on its own.
func (p *Pane) Toggle() (tea.Cmd, string) {
	if !p.hasImage {
		return nil, "ctrl+g switches raster charts between Kitty graphics and glyphs; this chart is text (add \"renderer\": \"raster\" to the document, or try flint-edit -example 8)"
	}
	p.modeFixed = true
	before := p.pic.Mode()
	cmd := p.pic.Toggle()
	switch after := p.pic.Mode(); {
	case after == before:
		return cmd, "this terminal did not report Kitty graphics support, so images stay as glyphs (FLINT_KITTY=1 forces Kitty graphics)"
	case after == picture.PictureKitty:
		return cmd, "image mode: Kitty graphics"
	default:
		return cmd, "image mode: glyphs"
	}
}

// Update handles the pane's own messages: the terminal's probe replies and the
// Kitty frames. Forward every message the host does not consume.
func (p *Pane) Update(msg tea.Msg) tea.Cmd {
	cmds := []tea.Cmd{p.pic.Update(msg)}
	if !p.modeFixed && p.pic.Mode() == picture.PictureGlyph && picture.KittySupported() == picture.KittyCapabilitySupported {
		cmds = append(cmds, p.pic.Toggle())
	}
	return tea.Batch(cmds...)
}

// View is the chart: the image's cells, or the text.
func (p *Pane) View() string {
	if p.hasImage {
		return strings.TrimRight(p.pic.View().Content, "\n")
	}
	return p.text
}
