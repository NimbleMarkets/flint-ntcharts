//go:build js && wasm

// Package clipboard stands in for github.com/atotto/clipboard, which has no
// js/wasm support. Copy and paste go through an in-memory buffer, and writes
// are mirrored to the browser clipboard when it is available.
package clipboard

import "syscall/js"

var buffer string

func ReadAll() (string, error) { return buffer, nil }

func WriteAll(text string) error {
	buffer = text
	if nav := js.Global().Get("navigator"); nav.Truthy() {
		if cb := nav.Get("clipboard"); cb.Truthy() {
			cb.Call("writeText", text)
		}
	}
	return nil
}

func Unsupported() bool { return false }
