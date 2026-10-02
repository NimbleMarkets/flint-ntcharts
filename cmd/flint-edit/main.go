// Command flint-edit is a live split-pane playground: edit a Flint chart spec
// (or a raw ntcharts-spec) with syntax highlighting in the left pane and watch
// it compile and render in the right pane on every keystroke. It shares the
// exact compile→build→render path used by flint-tui, so what you see here is
// what the renderer produces. The same playground runs in the browser; see
// demo/.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/internal/editor"
)

func main() {
	example := flag.Int("example", 1, fmt.Sprintf("start on built-in example `N` (1-%d); 8 is a raster chart", editor.Count()))
	flag.Parse()
	if *example < 1 || *example > editor.Count() {
		fmt.Fprintf(os.Stderr, "flint-edit: -example must be 1-%d\n", editor.Count())
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner, err := compile.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "flint-edit: compiler init: %v\n", err)
		os.Exit(1)
	}
	defer runner.Close(ctx)

	p := tea.NewProgram(editor.NewAt(runner, *example-1), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "flint-edit: %v\n", err)
		os.Exit(1)
	}
}
