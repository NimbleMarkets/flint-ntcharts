// Command flint-edit is a live split-pane playground: edit a Flint chart spec
// (or a raw ntcharts-spec) with syntax highlighting in the left pane and watch
// it compile and render in the right pane on every keystroke. It shares the
// exact compile→build→render path used by flint-tui, so what you see here is
// what the renderer produces. The same playground runs in the browser; see
// demo/.
package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/internal/editor"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner, err := compile.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "flint-edit: compiler init: %v\n", err)
		os.Exit(1)
	}
	defer runner.Close(ctx)

	p := tea.NewProgram(editor.New(runner), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "flint-edit: %v\n", err)
		os.Exit(1)
	}
}
