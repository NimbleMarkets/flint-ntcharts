// Command flint-tui is a persistent terminal chart window. Agents and scripts
// push flint inputs, compile envelopes, or ntcharts-spec documents via a
// watched file, stdin, or a unix socket; the newest document is rendered and
// re-fitted on every terminal resize.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/NimbleMarkets/flint-ntcharts/compile"
	"github.com/NimbleMarkets/flint-ntcharts/tui"
)

type config struct {
	file   string
	stdin  bool
	socket string
	poll   time.Duration
}

func parseConfig(args []string) (config, error) {
	var cfg config
	fs := flag.NewFlagSet("flint-tui", flag.ContinueOnError)
	fs.BoolVar(&cfg.stdin, "stdin", false, "read a JSON document stream from stdin")
	fs.StringVar(&cfg.socket, "listen", "", "listen on a unix socket `path` for document streams")
	fs.DurationVar(&cfg.poll, "poll", 250*time.Millisecond, "file poll interval")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	switch fs.NArg() {
	case 0:
	case 1:
		cfg.file = fs.Arg(0)
	default:
		return cfg, fmt.Errorf("at most one file argument, got %d", fs.NArg())
	}
	if cfg.file == "" && !cfg.stdin && cfg.socket == "" {
		return cfg, fmt.Errorf("no source: pass a file, --stdin, or --listen")
	}
	return cfg, nil
}

func run(cfg config) int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner, err := compile.New(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "flint-tui: compiler init: %v\n", err)
		return 1
	}
	defer runner.Close(ctx)

	opts := []tea.ProgramOption{tea.WithContext(ctx)}
	if cfg.stdin {
		// stdin carries data, not key input; use the TTY for keys if available.
		tty, err := os.Open("/dev/tty")
		if err == nil {
			defer tty.Close()
			opts = append(opts, tea.WithInput(tty))
		}
	}
	p := tea.NewProgram(tui.New(runner), opts...)

	send := func(msg tea.Msg) { p.Send(msg) }
	if cfg.file != "" {
		go tui.WatchFile(ctx, cfg.file, cfg.poll, send)
	}
	if cfg.stdin {
		// ReadStream's ctx cancellation only takes effect once the reader is
		// closed by its owner: it's blocked in dec.Decode() on os.Stdin,
		// which ctx cancellation alone does not unblock. That's acceptable
		// here because os.Stdin is owned by the process, not by us — process
		// exit (via p.Run() returning below) reclaims the fd and the
		// goroutine along with it. Callers passing a reader they must
		// close explicitly (e.g. a socket conn) close it themselves; see
		// ListenSocket in tui/sources.go.
		go tui.ReadStream(ctx, os.Stdin, "stdin", send)
	}
	if cfg.socket != "" {
		if err := tui.ListenSocket(ctx, cfg.socket, send); err != nil {
			fmt.Fprintf(os.Stderr, "flint-tui: listen: %v\n", err)
			return 1
		}
		defer os.Remove(cfg.socket)
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "flint-tui: %v\n", err)
		return 1
	}
	return 0
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "flint-tui: %v\nusage: flint-tui [--stdin] [--listen SOCK] [--poll 250ms] [chart.json]\n", err)
		os.Exit(2)
	}
	os.Exit(run(cfg))
}
