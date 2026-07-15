JAVY ?= bin/javy

.PHONY: wasm
wasm:
	cd js && npm run build
	$(JAVY) build -o compile/flint.wasm js/dist/flint-javy.js
	@ls -la compile/flint.wasm
