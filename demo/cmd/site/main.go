// Command site builds the browser demo into dist/: the flint-edit wasm, the
// go-booba terminal runtime, the booba-shim flint compiler bundle, and the
// page that loads them. Run it from the demo directory:
//
//	go run ./cmd/site
package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed index.html.tmpl
var page []byte

const out = "dist"

func main() {
	if err := os.RemoveAll(out); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		log.Fatal(err)
	}
	// booba-assets writes the runtime and a starter index.html, which the page
	// below replaces.
	run(nil, "go", "tool", "booba-assets", out)
	run(nil, "go", "tool", "booba-shim-assets", out, "--shim=flintchart")
	// The shim bundle normally comes from the booba-shim release in go.mod.
	// FLINTCHART_SHIM=<file> swaps in another build (make browser-shim writes
	// js/dist/flintchart-shim.mjs), to try compiler changes before a release.
	if override := os.Getenv("FLINTCHART_SHIM"); override != "" {
		data, err := os.ReadFile(override)
		if err != nil {
			log.Fatal(err)
		}
		dst := filepath.Join(out, "booba-shim", "flintchart", "flintchart-shim.js")
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Println("using", override, "for the flintchart shim")
	}
	run([]string{"GOOS=js", "GOARCH=wasm"}, "go", "build", "-o", filepath.Join(out, "app.wasm"), ".")
	if err := os.WriteFile(filepath.Join(out, "index.html"), page, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("built", out)
}

func run(env []string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	fmt.Println("+", name, args)
	if err := cmd.Run(); err != nil {
		log.Fatalf("%s: %v", name, err)
	}
}
