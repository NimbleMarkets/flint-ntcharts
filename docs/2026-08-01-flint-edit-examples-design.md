# flint-edit Example Switcher Design

**Date:** 2026-08-01
**Status:** Approved
**Scope:** `cmd/flint-edit` — multiple built-in example specs, switchable with
ctrl+1/2/3 (and alt+1/2/3).

## Summary

`flint-edit` currently seeds its editor with a single hard-coded Bar Chart
sample. This adds an ordered set of three named examples and key chords to load
them, replacing the editor buffer and re-rendering immediately.

## Examples

A new `cmd/flint-edit/examples.go` defines:

```go
type example struct {
    name string // shown on the status line, e.g. "Candlestick"
    src  string // full ChartAssemblyInput JSON
}

var examples = []example{...} // index 0 = key 1, etc.
```

| Slot | Name | Content |
| --- | --- | --- |
| 1 | Bar Chart | the existing `sample` const (revenue-by-month), moved into the list |
| 2 | Timeseries Line | adapted from `testdata/fixtures-terminal/line-temporal.json` |
| 3 | Candlestick | adapted from `testdata/fixtures-terminal/candlestick.json` |

Examples are string consts compiled into the binary. **Not** `go:embed` of
testdata (embed cannot reach above the package directory, and the demo should
not silently change when test fixtures do) and **not** runtime file loading
(would break an installed binary). "Adapted from" means the fixture's
`ChartAssemblyInput` content is copied in, with data/labels tweaked freely for
demo quality; no generated coupling.

The startup buffer stays example 1 — `newModel` seeds from `examples[0].src`,
and the standalone `sample` const is absorbed into `examples.go`.

## Keys

In `Update`'s `tea.KeyMsg` case, before the editor sees the key:

- `ctrl+1` / `ctrl+2` / `ctrl+3` and `alt+1` / `alt+2` / `alt+3` load the
  corresponding example.
- Digits with no example (e.g. `alt+9`) are ignored and fall through to the
  editor unchanged.

Loading an example: `ed.SetContent(src)`, cursor to end, `m.src = src`,
`m.gen++`, fire `renderCmd()`, and set the status line to
`loaded example: <name>` (pending state until the render lands).
Switching replaces the buffer without confirmation — flint-edit is a
playground; current edits are disposable.

Bubble Tea v2 requests kitty key disambiguation from the terminal by default,
so ctrl+digit is distinguishable wherever the terminal supports the protocol
(Ghostty, Kitty, WezTerm, newer iTerm2) with no opt-in code on our side.
alt+digit is ESC-prefixed and works in effectively all terminals — it is the
portable fallback, and the reason ctrl-only was rejected. In a terminal
without the protocol, ctrl chords simply never arrive: alt chords still work.

## UI

Title-bar hint becomes:

```
flint-edit   edit the spec, watch it render  ·  ctrl/alt+1·2·3 examples  ·  ctrl+c quit
```

## Testing

Extend `cmd/flint-edit/main_test.go` (same fake-compiler/renderNow harness):

- alt+2 / alt+3 key messages switch `ed.GetCurrentContent()` to the expected
  example source and return a non-nil render command with a bumped generation.
- An out-of-range digit chord leaves the buffer unchanged.
- Each example's `src` compiles cleanly through the real render path (guards
  against a typo'd example shipping broken).

Existing tests (`TestEditorHighlights`, render-path tests) are untouched.
