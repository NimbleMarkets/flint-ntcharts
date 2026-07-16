package tui

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func collector() (func(tea.Msg), chan tea.Msg) {
	ch := make(chan tea.Msg, 16)
	return func(m tea.Msg) { ch <- m }, ch
}

func waitInput(t *testing.T, ch chan tea.Msg) InputMsg {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case m := <-ch:
			if im, ok := m.(InputMsg); ok {
				return im
			}
		case <-deadline:
			t.Fatal("timed out waiting for InputMsg")
		}
	}
}

func TestReadStreamMultipleDocs(t *testing.T) {
	send, ch := collector()
	r := strings.NewReader(`{"type":"bar","width":10,"height":5,"data":{"series":[{"name":"a","values":[{"y":1}]}]}}
{"type":"bar","width":10,"height":5,"data":{"series":[{"name":"b","values":[{"y":2}]}]}}`)
	go ReadStream(context.Background(), r, "stdin", send)
	first := waitInput(t, ch)
	if !strings.Contains(string(first.Raw), `"a"`) {
		t.Fatalf("first doc wrong: %s", first.Raw)
	}
	second := waitInput(t, ch)
	if !strings.Contains(string(second.Raw), `"b"`) {
		t.Fatalf("second doc wrong: %s", second.Raw)
	}
}

func TestReadStreamPrettyPrintedDoc(t *testing.T) {
	send, ch := collector()
	r := strings.NewReader("{\n  \"type\": \"bar\",\n  \"width\": 10,\n  \"height\": 5,\n  \"data\": {\"series\": [{\"name\": \"a\", \"values\": [{\"y\": 1}]}]}\n}\n")
	go ReadStream(context.Background(), r, "stdin", send)
	got := waitInput(t, ch)
	if !strings.Contains(string(got.Raw), `"bar"`) {
		t.Fatalf("pretty doc not decoded: %s", got.Raw)
	}
}

func TestWatchFileDetectsChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "chart.json")
	if err := os.WriteFile(path, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	send, ch := collector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go WatchFile(ctx, path, 20*time.Millisecond, send)
	first := waitInput(t, ch)
	if !strings.Contains(string(first.Raw), `"v":1`) {
		t.Fatalf("initial read wrong: %s", first.Raw)
	}
	// mtime granularity: rewrite with different content and a nudge
	time.Sleep(30 * time.Millisecond)
	if err := os.WriteFile(path, []byte(`{"v":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	second := waitInput(t, ch)
	if !strings.Contains(string(second.Raw), `"v":2`) {
		t.Fatalf("change not detected: %s", second.Raw)
	}
}

func TestListenSocketDeliversDocs(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "flint.sock")
	send, ch := collector()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := ListenSocket(ctx, sock, send); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write([]byte(`{"type":"bar","width":10,"height":5,"data":{"series":[{"name":"s","values":[{"y":9}]}]}}` + "\n")); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	got := waitInput(t, ch)
	if got.Source == "" || !strings.Contains(string(got.Raw), `"s"`) {
		t.Fatalf("socket doc wrong: source=%q raw=%s", got.Source, got.Raw)
	}
}
