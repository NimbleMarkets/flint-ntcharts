package tui

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
)

// ReadStream decodes a sequence of JSON documents (NDJSON or pretty-printed,
// json.Decoder handles both) and sends each as an InputMsg. Returns on EOF,
// decode error, or ctx cancellation (via the reader being closed by its owner).
func ReadStream(ctx context.Context, r io.Reader, source string, send func(tea.Msg)) {
	dec := json.NewDecoder(r)
	for {
		var raw json.RawMessage
		err := dec.Decode(&raw)
		if err != nil {
			if !errors.Is(err, io.EOF) && ctx.Err() == nil {
				send(SourceErrMsg{Err: err})
			}
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		send(InputMsg{Raw: raw, Source: source})
	}
}

// WatchFile polls path every interval and sends the file's content whenever
// its mtime or size changes (including the initial read). Missing files are
// tolerated (the file may appear later or be mid-rewrite).
func WatchFile(ctx context.Context, path string, interval time.Duration, send func(tea.Msg)) {
	var lastMod time.Time
	var lastSize int64
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		if st, err := os.Stat(path); err == nil {
			if !st.ModTime().Equal(lastMod) || st.Size() != lastSize {
				lastMod, lastSize = st.ModTime(), st.Size()
				if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
					send(InputMsg{Raw: b, Source: "file:" + path})
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// ListenSocket creates a unix socket at path (removing any stale one) and
// serves each connection as a document stream. Returns an error only if the
// initial listen fails; per-connection errors go to send as SourceErrMsg.
func ListenSocket(ctx context.Context, path string, send func(tea.Msg)) error {
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				if ctx.Err() == nil {
					send(SourceErrMsg{Err: err})
				}
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				ReadStream(ctx, c, "socket:"+path, send)
			}(conn)
		}
	}()
	return nil
}
