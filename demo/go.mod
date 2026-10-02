module github.com/NimbleMarkets/flint-ntcharts/demo

go 1.26.8

require (
	github.com/NimbleMarkets/booba-shim v0.4.0
	github.com/NimbleMarkets/flint-ntcharts v0.2.0
	github.com/NimbleMarkets/go-booba v0.7.0
	github.com/NimbleMarkets/ntcharts/v2 v2.6.0
)

require (
	charm.land/bubbles/v2 v2.2.1 // indirect
	charm.land/bubbletea/v2 v2.0.10 // indirect
	charm.land/lipgloss/v2 v2.0.6 // indirect
	github.com/NimbleMarkets/pixterm v0.0.0-20260501211346-dc18ac6c1a0f // indirect
	github.com/alecthomas/chroma/v2 v2.26.1 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/charmbracelet/colorprofile v0.4.3 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20260928045949-bbf040aedf25 // indirect
	github.com/charmbracelet/x/ansi v0.11.8 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/disintegration/imaging v1.6.2 // indirect
	github.com/dlclark/regexp2/v2 v2.1.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-analyze/bulk v0.1.5 // indirect
	github.com/go-analyze/charts v0.6.1 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/ionut-t/goeditor v0.4.16 // indirect
	github.com/lrstanley/bubblezone/v2 v2.0.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.30 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/xo/terminfo v1.2.0 // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

replace github.com/NimbleMarkets/flint-ntcharts => ../

replace charm.land/bubbletea/v2 => github.com/neomantra/bubbletea/v2 v2.0.0-20260928192001-1b36865b418a

// atotto/clipboard (via bubbles' textinput) has no js/wasm support; see
// clipboard/. This module is a leaf, so the replaces do not reach consumers.
replace github.com/atotto/clipboard => ./clipboard

tool (
	github.com/NimbleMarkets/booba-shim/cmd/booba-shim-assets
	github.com/NimbleMarkets/go-booba/cmd/booba-assets
)
