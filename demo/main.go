//go:build js && wasm

// Command demo runs the flint-edit playground in the browser: the same model
// as the terminal tool, hosted by go-booba, compiling specs through the
// booba-shim flintchart bridge instead of the embedded wazero compiler.
package main

import (
	"fmt"

	booba "github.com/NimbleMarkets/go-booba"

	"github.com/NimbleMarkets/flint-ntcharts/internal/editor"
)

func main() {
	if err := booba.Run(editor.New(shimCompiler{})); err != nil {
		fmt.Println("flint-edit:", err)
	}
}
