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
