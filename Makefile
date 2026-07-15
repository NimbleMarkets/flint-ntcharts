JAVY ?= bin/javy

.PHONY: wasm
wasm: $(JAVY)
	cd js && npm run build
	$(JAVY) build -o compile/flint.wasm js/dist/flint-javy.js
	@ls -la compile/flint.wasm

$(JAVY):
	mkdir -p bin
	gh release download -R bytecodealliance/javy --pattern "javy-arm-macos-*.gz" -O bin/javy.gz --clobber
	gunzip -f bin/javy.gz
	chmod +x bin/javy
